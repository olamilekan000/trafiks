/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const trafiksIngressClassName = "trafiks"

// IngressReconciler reconciles Ingress resources and handles TLS termination
type IngressReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Logger       logr.Logger
	tlsConfigs   map[string]*tls.Config            // host -> TLS config
	ingressHosts map[types.NamespacedName][]string // ingress -> hosts for cleanup
	tlsMu        sync.RWMutex
	httpServer   *http.Server
	httpsServer  *http.Server
	started      bool
	startMu      sync.Mutex
}

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.WithValues("ingress", req.NamespacedName)

	ingress := &networkingv1.Ingress{}
	if err := r.Get(ctx, req.NamespacedName, ingress); err != nil {
		if apierrors.IsNotFound(err) {
			r.removeTLSConfigForIngress(req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !r.hasTrafiksIngressClass(ingress) {
		log.V(1).Info("Skipping Ingress - not using trafiks ingress class")
		return ctrl.Result{}, nil
	}

	// Start HTTP servers if not already started
	r.startMu.Lock()
	if !r.started {
		if err := r.StartHTTPServer(ctx); err != nil {
			r.startMu.Unlock()
			log.Error(fmt.Errorf("failed to start HTTP server: %v", err), "")
			return ctrl.Result{}, err
		}
		r.started = true
	}
	r.startMu.Unlock()

	// Process TLS configuration
	if len(ingress.Spec.TLS) > 0 {
		if err := r.loadTLSCertificates(ctx, ingress); err != nil {
			log.Error(fmt.Errorf("failed to load TLS certificates: %v", err), "")
			return ctrl.Result{}, err
		}
	}

	log.Info("Processed Ingress", "hosts", r.getIngressHosts(ingress))
	return ctrl.Result{}, nil
}

func (r *IngressReconciler) hasTrafiksIngressClass(ingress *networkingv1.Ingress) bool {
	if ingress.Spec.IngressClassName == nil {
		return false
	}
	return *ingress.Spec.IngressClassName == trafiksIngressClassName
}

// loadTLSCertificates loads TLS certificates from Ingress TLS secrets
func (r *IngressReconciler) loadTLSCertificates(ctx context.Context, ingress *networkingv1.Ingress) error {
	for _, tlsConfig := range ingress.Spec.TLS {
		secretName := tlsConfig.SecretName
		if secretName == "" {
			continue
		}

		secret := &corev1.Secret{}
		secretKey := types.NamespacedName{
			Namespace: ingress.Namespace,
			Name:      secretName,
		}

		if err := r.Get(ctx, secretKey, secret); err != nil {
			return fmt.Errorf("failed to get TLS secret %s/%s: %w", ingress.Namespace, secretName, err)
		}

		certBytes := secret.Data["tls.crt"]
		keyBytes := secret.Data["tls.key"]

		if len(certBytes) == 0 || len(keyBytes) == 0 {
			return fmt.Errorf("TLS secret %s/%s missing tls.crt or tls.key", ingress.Namespace, secretName)
		}

		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return fmt.Errorf("failed to parse TLS certificate: %w", err)
		}

		ingressKey := types.NamespacedName{
			Namespace: ingress.Namespace,
			Name:      ingress.Name,
		}

		r.tlsMu.Lock()
		oldHosts := r.ingressHosts[ingressKey]
		for _, oldHost := range oldHosts {
			delete(r.tlsConfigs, oldHost)
		}

		newHosts := []string{}
		for _, host := range tlsConfig.Hosts {
			r.tlsConfigs[host] = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
			newHosts = append(newHosts, host)
		}
		r.ingressHosts[ingressKey] = newHosts
		r.tlsMu.Unlock()

		r.Logger.Info("Loaded TLS certificate", "secret", secretName, "hosts", tlsConfig.Hosts)
	}

	return nil
}

// removeTLSConfigForIngress removes TLS configs for hosts belonging to the deleted Ingress
func (r *IngressReconciler) removeTLSConfigForIngress(ingressKey types.NamespacedName) {
	r.tlsMu.Lock()
	defer r.tlsMu.Unlock()

	hosts, exists := r.ingressHosts[ingressKey]
	if !exists {
		r.Logger.V(1).Info("No TLS configs found for Ingress", "ingress", ingressKey)
		return
	}

	for _, host := range hosts {
		delete(r.tlsConfigs, host)
		r.Logger.V(1).Info("Removed TLS config for host", "host", host, "ingress", ingressKey)
	}

	delete(r.ingressHosts, ingressKey)
	r.Logger.Info("Removed TLS configs for Ingress", "ingress", ingressKey, "hosts", hosts)
}

func (r *IngressReconciler) getTLSConfig(host string) *tls.Config {
	r.tlsMu.RLock()
	defer r.tlsMu.RUnlock()
	return r.tlsConfigs[host]
}

func (r *IngressReconciler) StartHTTPServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", r.handleRequest)

	r.httpServer = &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	r.httpsServer = &http.Server{
		Addr:    ":443",
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				tlsConfig := r.getTLSConfig(clientHello.ServerName)
				if tlsConfig == nil || len(tlsConfig.Certificates) == 0 {
					return nil, fmt.Errorf("no certificate for host: %s", clientHello.ServerName)
				}
				return &tlsConfig.Certificates[0], nil
			},
		},
	}

	go func() {
		r.Logger.Info("Starting HTTP server on :80")
		if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.Logger.Error(fmt.Errorf("HTTP server error: %v", err), "")
		}
	}()

	go func() {
		r.Logger.Info("Starting HTTPS server on :443")
		if err := r.httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			r.Logger.Error(fmt.Errorf("HTTPS server error: %v", err), "")
		}
	}()

	return nil
}

func (r *IngressReconciler) handleRequest(w http.ResponseWriter, req *http.Request) {
	r.Logger.Info("Handling request", "host", req.Host, "path", req.URL.Path, "method", req.Method)

	host := req.Host
	path := req.URL.Path

	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	r.Logger.Info("Looking for matching Ingress", "host", host, "path", path)

	ingress, rule := r.findMatchingIngress(context.Background(), host, path)
	if ingress == nil {
		r.Logger.Info("No matching Ingress found", "host", host, "path", path)
		http.NotFound(w, req)
		return
	}

	r.Logger.Info("Found matching Ingress", "ingress", ingress.Name, "namespace", ingress.Namespace)

	backend := r.getBackendService(ingress, rule)
	if backend == nil {
		r.Logger.Info("Failed to get backend service from Ingress", "ingress", ingress.Name)
		http.NotFound(w, req)
		return
	}

	r.Logger.Info("Resolved backend service", "service", backend.Name, "namespace", backend.Namespace, "port", backend.Port)

	originalHost := host
	if idx := strings.Index(originalHost, ":"); idx != -1 {
		originalHost = originalHost[:idx]
	}

	targetURL := r.buildTargetURL(context.Background(), backend, path)

	if req.URL.RawQuery != "" {
		targetURL += "?" + req.URL.RawQuery
	}

	r.Logger.Info("Forwarding request", "targetURL", targetURL, "originalHost", originalHost, "scheme", strings.Split(targetURL, "://")[0])

	r.forwardRequest(w, req, targetURL, originalHost)
}

func (r *IngressReconciler) findMatchingIngress(ctx context.Context, host, path string) (*networkingv1.Ingress, *networkingv1.IngressRule) {
	ingressList := &networkingv1.IngressList{}
	if err := r.List(ctx, ingressList); err != nil {
		r.Logger.Error(fmt.Errorf("failed to list Ingresses: %v", err), "")
		return nil, nil
	}

	r.Logger.Info("Found Ingresses", "count", len(ingressList.Items))

	for _, ingress := range ingressList.Items {
		hasClass := r.hasTrafiksIngressClass(&ingress)
		r.Logger.V(1).Info("Checking Ingress", "name", ingress.Name, "namespace", ingress.Namespace, "hasTrafiksClass", hasClass)

		if !hasClass {
			continue
		}

		for _, rule := range ingress.Spec.Rules {
			r.Logger.V(1).Info("Checking rule", "ingress", ingress.Name, "ruleHost", rule.Host, "requestHost", host, "match", rule.Host == host)

			if rule.Host == host && rule.HTTP != nil {
				for _, pathRule := range rule.HTTP.Paths {
					pathType := networkingv1.PathTypePrefix
					if pathRule.PathType != nil {
						pathType = *pathRule.PathType
					}
					matches := r.pathMatches(path, pathRule.Path, pathType)
					r.Logger.V(1).Info("Checking path", "rulePath", pathRule.Path, "requestPath", path, "pathType", pathType, "matches", matches)

					if matches {
						r.Logger.Info("Matched Ingress rule", "ingress", ingress.Name, "host", host, "path", path)
						return &ingress, &rule
					}
				}
			}
		}
	}

	r.Logger.Info("No matching Ingress found after checking all", "host", host, "path", path)
	return nil, nil
}

// pathMatches checks if request path matches rule path
func (r *IngressReconciler) pathMatches(requestPath, rulePath string, pathType networkingv1.PathType) bool {
	switch pathType {
	case networkingv1.PathTypeExact:
		return requestPath == rulePath
	case networkingv1.PathTypePrefix:
		return strings.HasPrefix(requestPath, rulePath) ||
			strings.HasPrefix(requestPath, path.Clean(rulePath)+"/")
	case networkingv1.PathTypeImplementationSpecific:
		return strings.HasPrefix(requestPath, rulePath)
	default:
		return false
	}
}

// BackendServiceInfo holds backend service information
type BackendServiceInfo struct {
	Namespace string
	Name      string
	Port      int32
}

// getBackendService extracts backend service from Ingress rule
func (r *IngressReconciler) getBackendService(ingress *networkingv1.Ingress, rule *networkingv1.IngressRule) *BackendServiceInfo {
	if rule.HTTP == nil || len(rule.HTTP.Paths) == 0 {
		return nil
	}

	path := rule.HTTP.Paths[0]
	if path.Backend.Service == nil {
		return nil
	}

	port := int32(80)
	if path.Backend.Service.Port.Number != 0 {
		port = path.Backend.Service.Port.Number
	} else if path.Backend.Service.Port.Name != "" {
		resolvedPort, err := r.resolvePortName(context.Background(), ingress.Namespace, path.Backend.Service.Name, path.Backend.Service.Port.Name)
		if err != nil {
			r.Logger.Error(fmt.Errorf("failed to resolve port name for service %s/%s port %s: %v", ingress.Namespace, path.Backend.Service.Name, path.Backend.Service.Port.Name, err), "")
			return nil
		}
		port = resolvedPort
	}

	return &BackendServiceInfo{
		Namespace: ingress.Namespace,
		Name:      path.Backend.Service.Name,
		Port:      port,
	}
}

func (r *IngressReconciler) resolvePortName(ctx context.Context, namespace, serviceName, portName string) (int32, error) {
	svc := &corev1.Service{}
	key := types.NamespacedName{Namespace: namespace, Name: serviceName}

	if err := r.Get(ctx, key, svc); err != nil {
		return 0, fmt.Errorf("failed to get service: %w", err)
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == portName {
			return port.Port, nil
		}
	}

	return 0, fmt.Errorf("port %s not found in service %s/%s", portName, namespace, serviceName)
}

func (r *IngressReconciler) isRunningInCluster() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

func (r *IngressReconciler) buildTargetURL(ctx context.Context, backend *BackendServiceInfo, path string) string {
	if !r.isRunningInCluster() {
		r.Logger.Info("Running locally, using localhost", "service", backend.Name, "port", backend.Port,
			"hint", fmt.Sprintf("Ensure port-forward is running: kubectl port-forward -n %s svc/%s %d:%d",
				backend.Namespace, backend.Name, backend.Port, backend.Port))
		return fmt.Sprintf("http://localhost:%d%s", backend.Port, path)
	}

	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s",
		backend.Name, backend.Namespace, backend.Port, path)
}

func (r *IngressReconciler) forwardRequest(w http.ResponseWriter, req *http.Request, targetURL string, originalHost string) {
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, targetURL, req.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	for k, v := range req.Header {
		if strings.ToLower(k) == "host" {
			continue
		}
		newReq.Header[k] = v
	}

	newReq.Host = originalHost
	newReq.Header.Set("Host", originalHost)
	newReq.Header.Set("X-Forwarded-Host", req.Host)

	clientIP := r.getClientIP(req)
	newReq.Header.Set("X-Forwarded-For", clientIP)
	newReq.Header.Set("X-Real-IP", clientIP)

	tlsConfig := r.getTLSConfig(originalHost)
	hasTLS := req.TLS != nil || (tlsConfig != nil && len(tlsConfig.Certificates) > 0)

	scheme := "http"
	if hasTLS {
		scheme = "https"
	}

	newReq.Header.Set("X-Forwarded-Proto", scheme)
	newReq.Header.Set("X-Scheme", scheme)

	if hasTLS {
		r.Logger.V(1).Info("Setting X-Forwarded-Proto to https", "reason", map[string]bool{
			"requestTLS":      req.TLS != nil,
			"hasCertificates": tlsConfig != nil && len(tlsConfig.Certificates) > 0,
		})
	}

	r.Logger.V(1).Info("Forwarding request with Host header", "host", originalHost, "targetURL", targetURL)

	clientTLSConfig := &tls.Config{}

	if tlsConfig != nil && len(tlsConfig.Certificates) > 0 {
		clientTLSConfig.Certificates = tlsConfig.Certificates
		r.Logger.V(1).Info("Using certificate from Ingress secret for client TLS", "host", originalHost)
	} else {
		r.Logger.V(1).Info("No TLS config found for host, using insecure skip", "host", originalHost)
		clientTLSConfig.InsecureSkipVerify = true
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLSConfig,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			r.Logger.Info("Redirect detected", "from", via[len(via)-1].URL, "to", req.URL)
			return nil
		},
	}
	resp, err := client.Do(newReq)
	if err != nil {
		r.Logger.Error(fmt.Errorf("failed to reach backend %s %s: %v", newReq.Method, targetURL, err), "")
		http.Error(w, fmt.Sprintf("Failed to reach backend: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// Copy status and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (r *IngressReconciler) getClientIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	remoteAddr := req.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		remoteAddr = remoteAddr[:idx]
	}

	if strings.HasPrefix(remoteAddr, "[") && strings.HasSuffix(remoteAddr, "]") {
		remoteAddr = remoteAddr[1 : len(remoteAddr)-1]
	}

	return remoteAddr
}

func (r *IngressReconciler) getIngressHosts(ingress *networkingv1.Ingress) []string {
	var hosts []string
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	return hosts
}

func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.tlsConfigs = make(map[string]*tls.Config)
	r.ingressHosts = make(map[types.NamespacedName][]string)

	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
