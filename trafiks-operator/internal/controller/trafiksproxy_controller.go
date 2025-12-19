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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	proxyv1 "github.com/trafiks/operator/api/v1"
)

const trafiksProxyFinalizer = "trafiksproxy.proxy.trafiks.io/finalizer"

// TrafiksProxyReconciler reconciles a TrafiksProxy object
type TrafiksProxyReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	APIClient TrafiksAPIClient
	Logger    logr.Logger
}

// +kubebuilder:rbac:groups=proxy.trafiks.io,resources=trafiksproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=proxy.trafiks.io,resources=trafiksproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=proxy.trafiks.io,resources=trafiksproxies/finalizers,verbs=update
// +kubebuilder:rbac:groups=proxy.trafiks.io,resources=trafiksbackends,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TrafiksProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger

	proxy := &proxyv1.TrafiksProxy{}
	if err := r.Get(ctx, req.NamespacedName, proxy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch TrafiksProxy")
		return ctrl.Result{}, err
	}

	if !proxy.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, proxy, log)
	}

	if !containsString(proxy.Finalizers, trafiksProxyFinalizer) {
		proxy.Finalizers = append(proxy.Finalizers, trafiksProxyFinalizer)
		if err := r.Update(ctx, proxy); err != nil {
			log.Error(err, "unable to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if proxy.Status.Conditions == nil {
		proxy.Status.Conditions = []metav1.Condition{}
	}

	ingressValidation := r.validateIngressSync(ctx, proxy)
	if !ingressValidation.shouldReconcile {
		if err := r.updateStatus(ctx, req.NamespacedName, proxy.Status); err != nil {
			log.Error(err, "unable to update TrafiksProxy status")
			return ctrl.Result{}, err
		}
		requeueAfter := ingressValidation.requeueAfter
		if requeueAfter == 0 {
			requeueAfter = 30 * time.Second
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	err := r.getConfiguredAPIClient(ctx, proxy)
	if err != nil {
		SetCondition(&proxy.Status.Conditions, "BackendReachable", metav1.ConditionFalse, "BackendNotFound",
			fmt.Sprintf("TrafiksBackend not found or not ready: %v", err), proxy.Generation)
		SetCondition(&proxy.Status.Conditions, "Ready", metav1.ConditionFalse, "BackendNotFound",
			"Cannot sync service: TrafiksBackend not available", proxy.Generation)
		if err := r.updateStatus(ctx, req.NamespacedName, proxy.Status); err != nil {
			log.Error(err, "unable to update TrafiksProxy status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	targetURL, resolvedPortName, err := r.resolveKubernetesService(ctx, &proxy.Spec.Kubernetes, proxy.Spec.Scheme)
	if err != nil {
		SetCondition(&proxy.Status.Conditions, "ServiceResolved", metav1.ConditionFalse, "ServiceNotFound",
			fmt.Sprintf("Kubernetes service not found: %v", err), proxy.Generation)

		SetCondition(&proxy.Status.Conditions, "Ready", metav1.ConditionFalse, "ServiceNotFound",
			"Cannot resolve Kubernetes service endpoint", proxy.Generation)

		if err := r.updateStatus(ctx, req.NamespacedName, proxy.Status); err != nil {
			log.Error(err, "unable to update TrafiksProxy status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	SetCondition(&proxy.Status.Conditions, "ServiceResolved", metav1.ConditionTrue, "ServiceResolved",
		fmt.Sprintf("Resolved Kubernetes service endpoint: %s", targetURL), proxy.Generation)

	projectName := proxy.Spec.ProjectName

	projectUID, err := r.findOrGetProject(ctx, projectName, proxy)
	if err != nil {
		SetCondition(&proxy.Status.Conditions, "Ready", metav1.ConditionFalse, "ProjectError",
			fmt.Sprintf("Failed to get project: %v", err), proxy.Generation)
		if err := r.updateStatus(ctx, req.NamespacedName, proxy.Status); err != nil {
			log.Error(err, "unable to update TrafiksProxy status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	serviceUID, err := r.syncServiceToTrafiks(ctx, projectUID, proxy, targetURL, resolvedPortName)
	if err != nil {
		SetCondition(&proxy.Status.Conditions, "Synced", metav1.ConditionFalse, "SyncFailed",
			fmt.Sprintf("Failed to sync service: %v", err), proxy.Generation)
		SetCondition(&proxy.Status.Conditions, "Ready", metav1.ConditionFalse, "SyncFailed",
			"Service sync failed", proxy.Generation)
		if err := r.updateStatus(ctx, req.NamespacedName, proxy.Status); err != nil {
			log.Error(err, "unable to update TrafiksProxy status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	proxy.Status.ServiceUID = serviceUID
	proxy.Status.ProjectUID = projectUID
	proxy.Status.ProjectName = projectName
	now := metav1.Now()
	proxy.Status.LastSyncTime = &now

	SetCondition(&proxy.Status.Conditions, "Synced", metav1.ConditionTrue, "Synced",
		"Service successfully synced to Trafiks backend", proxy.Generation)
	SetCondition(&proxy.Status.Conditions, "Ready", metav1.ConditionTrue, "Ready",
		"TrafiksProxy is ready and synced", proxy.Generation)

	if err := r.updateStatus(ctx, req.NamespacedName, proxy.Status); err != nil {
		log.Error(err, "unable to update TrafiksProxy status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *TrafiksProxyReconciler) getConfiguredAPIClient(ctx context.Context, proxy *proxyv1.TrafiksProxy) error {
	backendRef := proxy.Spec.BackendRef
	backendNamespace := backendRef.Namespace
	if backendNamespace == "" {
		backendNamespace = proxy.Namespace
	}

	baseURL, apiKey, err := r.getTrafiksBackendCredentials(ctx, backendNamespace, backendRef.Name)
	if err != nil {
		return nil
	}

	r.APIClient = r.APIClient.SetBaseURL(baseURL).SetAPIKey(apiKey)

	return nil
}

func (r *TrafiksProxyReconciler) getTrafiksBackendCredentials(ctx context.Context, namespace, name string) (string, string, error) {
	backend := &proxyv1.TrafiksBackend{}
	backendKey := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := r.Get(ctx, backendKey, backend); err != nil {
		return "", "", fmt.Errorf("failed to get TrafiksBackend %s/%s: %w", namespace, name, err)
	}

	readyCondition := findCondition(backend.Status.Conditions, "Ready")
	if readyCondition == nil || readyCondition.Status != metav1.ConditionTrue {
		return "", "", fmt.Errorf("TrafiksBackend %s is not ready", backend.Name)
	}

	secretRef := backend.Spec.SecretRef
	secretNamespace := secretRef.Namespace
	if secretNamespace == "" {
		secretNamespace = backend.Namespace
	}

	baseURLKey := backend.Spec.BaseURLKey
	if baseURLKey == "" {
		baseURLKey = "baseURL"
	}
	apiKeyKey := backend.Spec.APIKeyKey
	if apiKeyKey == "" {
		apiKeyKey = "apiKey"
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: secretNamespace,
		Name:      secretRef.Name,
	}

	if err := r.Get(ctx, secretKey, secret); err != nil {
		return "", "", fmt.Errorf("failed to get secret: %w", err)
	}

	baseURLBytes, baseURLExists := secret.Data[baseURLKey]
	apiKeyBytes, apiKeyExists := secret.Data[apiKeyKey]

	if !baseURLExists || !apiKeyExists {
		return "", "", fmt.Errorf("secret missing required keys: baseURL or apiKey")
	}

	baseURL := string(baseURLBytes)
	apiKey := string(apiKeyBytes)

	if baseURL == "" || apiKey == "" {
		return "", "", fmt.Errorf("secret contains empty values for baseURL or apiKey")
	}

	return baseURL, apiKey, nil
}

func (r *TrafiksProxyReconciler) resolveKubernetesService(ctx context.Context, k8sConfig *proxyv1.KubernetesProxyConfig, scheme string) (string, string, error) {
	svc := &corev1.Service{}
	svcKey := client.ObjectKey{
		Namespace: k8sConfig.Namespace,
		Name:      k8sConfig.ServiceName,
	}

	if err := r.Get(ctx, svcKey, svc); err != nil {
		return "", "", fmt.Errorf("failed to get service %s/%s: %w", k8sConfig.Namespace, k8sConfig.ServiceName, err)
	}

	port, portName, err := r.resolvePort(svc, k8sConfig.ServicePortName)
	if err != nil {
		return "", "", err
	}

	targetURL := fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", scheme, k8sConfig.ServiceName, k8sConfig.Namespace, port)

	return targetURL, portName, nil
}

func (r *TrafiksProxyReconciler) resolvePort(svc *corev1.Service, portName string) (int32, string, error) {
	if portName != "" {
		for _, port := range svc.Spec.Ports {
			if port.Name == portName {
				return port.Port, port.Name, nil
			}
		}
		return 0, "", fmt.Errorf("port name %s not found in service %s/%s", portName, svc.Namespace, svc.Name)
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == "http" {
			return port.Port, port.Name, nil
		}
	}

	for _, port := range svc.Spec.Ports {
		if port.Name != "" {
			return port.Port, port.Name, nil
		}
	}

	return 0, "", fmt.Errorf("no ports with names found in service %s/%s ServicePortName requires a port name", svc.Namespace, svc.Name)
}

func (r *TrafiksProxyReconciler) findOrGetProject(ctx context.Context, projectName string, proxy *proxyv1.TrafiksProxy) (string, error) {
	if proxy.Status.ProjectUID != "" {
		project, err := r.APIClient.GetProject(ctx, proxy.Status.ProjectUID)
		if err == nil && project != nil {
			if name, ok := project["name"].(string); ok && name == projectName {
				return proxy.Status.ProjectUID, nil
			}
		}

		proxy.Status.ProjectUID = ""
	}

	projects, err := r.APIClient.ListProjects(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	for _, project := range projects {
		if name, ok := project["name"].(string); ok && name == projectName {
			if uid, ok := project["uid"].(string); ok {
				return uid, nil
			}
		}
	}

	project, err := r.APIClient.CreateProject(ctx, projectName, "")
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	if uid, ok := project["uid"].(string); ok {
		return uid, nil
	}

	return "", fmt.Errorf("created project but no UID returned")
}

func (r *TrafiksProxyReconciler) syncServiceToTrafiks(ctx context.Context, projectUID string, proxy *proxyv1.TrafiksProxy, targetURL string, resolvedPortName string) (string, error) {
	serviceData, err := r.buildServiceRequest(ctx, proxy, targetURL, resolvedPortName)
	if err != nil {
		return "", fmt.Errorf("failed to build service request: %w", err)
	}

	if proxy.Status.ServiceUID != "" {
		updatedService, err := r.APIClient.UpdateService(ctx, projectUID, proxy.Status.ServiceUID, serviceData)
		if err != nil {
			return "", fmt.Errorf("failed to update service: %w", err)
		}

		if uid, ok := updatedService["uid"].(string); ok {
			return uid, nil
		}
		return proxy.Status.ServiceUID, nil
	}

	service, err := r.APIClient.CreateService(ctx, projectUID, serviceData)
	if err != nil {
		return "", fmt.Errorf("failed to create service: %w", err)
	}

	if uid, ok := service["uid"].(string); ok {
		return uid, nil
	}

	return "", fmt.Errorf("created service but no UID returned")
}

func (r *TrafiksProxyReconciler) buildServiceRequest(ctx context.Context, proxy *proxyv1.TrafiksProxy, targetURL string, resolvedPortName string) (map[string]interface{}, error) {
	scheme := proxy.Spec.Scheme

	serviceData := map[string]interface{}{
		"Source":           "kubernetes",
		"Scheme":           scheme,
		"ProxyURL":         proxy.Spec.ProxyURL,
		"TargetBackendURL": targetURL,
	}

	if scheme == "https" {
		certPEM, keyPEM, certResolver, err := r.getTLSCertificate(ctx, proxy)
		if err != nil {
			r.Logger.Info(fmt.Sprintf("failed to get TLS certificate, will use self-signed: %v", err))
		}

		if certPEM != "" && keyPEM != "" {
			serviceData["TLSCertificate"] = certPEM
			serviceData["TLSKey"] = keyPEM
			serviceData["TLSCertResolver"] = certResolver
		}
	}

	if proxy.Spec.Cache != nil {
		serviceData["CacheEnabled"] = proxy.Spec.Cache.Enabled
		serviceData["CacheTTL"] = proxy.Spec.Cache.TTL
	}

	config := make(map[string]interface{})

	kubernetesConfig := map[string]interface{}{
		"namespace":    proxy.Spec.Kubernetes.Namespace,
		"service_name": proxy.Spec.Kubernetes.ServiceName,
	}

	portName := proxy.Spec.Kubernetes.ServicePortName
	if portName == "" {
		portName = resolvedPortName
	}
	if portName != "" {
		kubernetesConfig["service_port_name"] = portName
	}

	if len(proxy.Spec.Kubernetes.Selector) > 0 {
		kubernetesConfig["selector"] = proxy.Spec.Kubernetes.Selector
	}
	config["kubernetes"] = kubernetesConfig

	if proxy.Spec.Headers != nil {
		headersConfig := make(map[string]interface{})
		if len(proxy.Spec.Headers.Remove) > 0 {
			headersConfig["remove"] = proxy.Spec.Headers.Remove
		}
		if len(proxy.Spec.Headers.Add) > 0 {
			headersConfig["add"] = proxy.Spec.Headers.Add
		}
		if len(headersConfig) > 0 {
			config["headers"] = headersConfig
		}
	}

	if proxy.Spec.QueryParams != nil && len(proxy.Spec.QueryParams.Remove) > 0 {
		config["query_params"] = map[string]interface{}{
			"remove": proxy.Spec.QueryParams.Remove,
		}
	}

	if proxy.Spec.HTTPSRedirect != nil {
		config["https_redirect"] = *proxy.Spec.HTTPSRedirect
	}

	if len(config) > 0 {
		serviceData["Configuration"] = config
	}

	return serviceData, nil
}

func (r *TrafiksProxyReconciler) updateStatus(ctx context.Context, nn types.NamespacedName, status proxyv1.TrafiksProxyStatus) error {
	proxy := &proxyv1.TrafiksProxy{}
	return updateStatusWithRetry(ctx, r.Client, nn, proxy, func(obj client.Object) {
		proxy := obj.(*proxyv1.TrafiksProxy)
		proxy.Status = status
	})
}

func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// ingressValidationResult holds the result of Ingress validation
type ingressValidationResult struct {
	shouldReconcile bool
	requeueAfter    time.Duration
}

func (r *TrafiksProxyReconciler) validateIngressSync(ctx context.Context, proxy *proxyv1.TrafiksProxy) ingressValidationResult {
	if proxy.Status.IngressRef == nil {
		return r.discoverIngress(ctx, proxy)
	}

	return r.validateTrackedIngress(ctx, proxy)
}

func (r *TrafiksProxyReconciler) discoverIngress(ctx context.Context, proxy *proxyv1.TrafiksProxy) ingressValidationResult {
	ingressList := &networkingv1.IngressList{}
	if err := r.List(ctx, ingressList); err != nil {
		return ingressValidationResult{shouldReconcile: true}
	}

	for _, ingress := range ingressList.Items {
		annotationProxyURL := ingress.Annotations["trafiks.io/proxy-url"]
		if annotationProxyURL == proxy.Spec.ProxyURL {
			proxy.Status.IngressRef = &proxyv1.IngressReference{
				Name:      ingress.Name,
				Namespace: ingress.Namespace,
			}
			r.Logger.Info("Discovered Ingress, tracking in status",
				"ingress", ingress.Name,
				"proxyURL", proxy.Spec.ProxyURL)
			return ingressValidationResult{shouldReconcile: true}
		}
	}

	return ingressValidationResult{shouldReconcile: true}
}

// validateTrackedIngress validates the tracked Ingress exists and matches proxyURL
func (r *TrafiksProxyReconciler) validateTrackedIngress(ctx context.Context, proxy *proxyv1.TrafiksProxy) ingressValidationResult {
	ingress := &networkingv1.Ingress{}
	ingressKey := types.NamespacedName{
		Name:      proxy.Status.IngressRef.Name,
		Namespace: proxy.Status.IngressRef.Namespace,
	}

	if err := r.Get(ctx, ingressKey, ingress); err != nil {
		if apierrors.IsNotFound(err) {
			proxy.Status.IngressRef = nil
			SetCondition(&proxy.Status.Conditions, "Ready", metav1.ConditionFalse, "IngressNotFound",
				fmt.Sprintf("Tracked Ingress %s/%s not found", ingressKey.Namespace, ingressKey.Name), proxy.Generation)
			return ingressValidationResult{
				shouldReconcile: false,
				requeueAfter:    30 * time.Second,
			}
		}

		r.Logger.Error(err, "unable to fetch tracked Ingress")
		return ingressValidationResult{shouldReconcile: true}
	}

	annotationProxyURL := ingress.Annotations["trafiks.io/proxy-url"]
	if annotationProxyURL == "" {
		return ingressValidationResult{shouldReconcile: true}
	}

	if annotationProxyURL != proxy.Spec.ProxyURL {
		SetCondition(&proxy.Status.Conditions, "Ready", metav1.ConditionFalse, "ProxyURLMismatch",
			fmt.Sprintf("Ingress annotation (%s) does not match TrafiksProxy.spec.proxyURL %s. Please update spec.proxyURL to match",
				annotationProxyURL, proxy.Spec.ProxyURL), proxy.Generation)
		return ingressValidationResult{
			shouldReconcile: false,
			requeueAfter:    1 * time.Minute,
		}
	}

	r.clearMismatchCondition(proxy)
	return ingressValidationResult{shouldReconcile: true}
}

func (r *TrafiksProxyReconciler) clearMismatchCondition(proxy *proxyv1.TrafiksProxy) {
	conditions := proxy.Status.Conditions
	filtered := make([]metav1.Condition, 0, len(conditions))
	for _, condition := range conditions {
		if condition.Type != "ProxyURLMismatch" {
			filtered = append(filtered, condition)
		}
	}
	proxy.Status.Conditions = filtered
}

// getTLSCertificateFromIngress extracts TLS certificate from the tracked Ingress
// Returns certificate PEM, key PEM, and error
func (r *TrafiksProxyReconciler) getTLSCertificateFromIngress(ctx context.Context, proxy *proxyv1.TrafiksProxy) (string, string, error) {
	log := r.Logger
	if proxy.Status.IngressRef == nil {
		return "", "", fmt.Errorf("no Ingress reference found - Ingress must exist and have trafiks.io/proxy-url annotation matching proxyURL")
	}

	ingress := &networkingv1.Ingress{}
	ingressKey := types.NamespacedName{
		Name:      proxy.Status.IngressRef.Name,
		Namespace: proxy.Status.IngressRef.Namespace,
	}

	if err := r.Get(ctx, ingressKey, ingress); err != nil {
		return "", "", fmt.Errorf("failed to get Ingress %s/%s: %w", ingressKey.Namespace, ingressKey.Name, err)
	}

	for _, tls := range ingress.Spec.TLS {
		for _, host := range tls.Hosts {
			if host != proxy.Spec.ProxyURL {
				continue
			}

			secretName := tls.SecretName
			if secretName == "" {
				return "", "", fmt.Errorf("ingress TLS entry for %s has no secretName", proxy.Spec.ProxyURL)
			}

			certPEM, keyPEM, err := r.getCertificateFromSecret(ctx, ingress.Namespace, secretName)
			if err != nil {
				return "", "", fmt.Errorf("failed to get certificate from secret %s/%s: %w", ingress.Namespace, secretName, err)
			}

			log.Info("Extracted TLS certificate from Ingress",
				"ingress", ingress.Name,
				"secret", secretName,
				"proxyURL", proxy.Spec.ProxyURL)

			return certPEM, keyPEM, nil
		}
	}

	return "", "", fmt.Errorf("no TLS configuration found in Ingress for proxyURL %s", proxy.Spec.ProxyURL)
}

// getCertificateFromSecret retrieves TLS certificate and key from a Kubernetes Secret
func (r *TrafiksProxyReconciler) getCertificateFromSecret(ctx context.Context, namespace, secretName string) (string, string, error) {
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Name:      secretName,
		Namespace: namespace,
	}

	if err := r.Get(ctx, secretKey, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return "", "", fmt.Errorf("TLS secret %s/%s not found - cert-manager may not have created it yet", namespace, secretName)
		}
		return "", "", fmt.Errorf("failed to get TLS secret %s/%s: %w", namespace, secretName, err)
	}

	certBytes := secret.Data["tls.crt"]
	keyBytes := secret.Data["tls.key"]

	if len(certBytes) == 0 {
		return "", "", fmt.Errorf("secret %s/%s missing tls.crt", namespace, secretName)
	}

	if len(keyBytes) == 0 {
		return "", "", fmt.Errorf("secret %s/%s missing tls.key", namespace, secretName)
	}

	return string(certBytes), string(keyBytes), nil
}

// getTLSCertificate retrieves TLS certificate based on tlsCertResolver
// Returns certificate PEM, key PEM, resolver type, and error
func (r *TrafiksProxyReconciler) getTLSCertificate(ctx context.Context, proxy *proxyv1.TrafiksProxy) (string, string, string, error) {
	certResolver := proxy.Spec.TLSCertResolver
	if certResolver == "" {
		certResolver = "selfsigned"
	}

	certPEM, keyPEM, err := r.getTLSCertificateFromIngress(ctx, proxy)
	if err != nil {
		return "", "", certResolver, fmt.Errorf("failed to get certificate from Ingress: %w", err)
	}

	return certPEM, keyPEM, certResolver, nil
}

func (r *TrafiksProxyReconciler) ingressToTrafiksProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return []reconcile.Request{}
	}

	log := r.Logger
	proxyList := &proxyv1.TrafiksProxyList{}
	if err := r.List(ctx, proxyList); err != nil {
		log.Error(err, "unable to list TrafiksProxy resources")
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for _, proxy := range proxyList.Items {
		if r.shouldReconcileProxyForIngress(ingress, &proxy) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      proxy.Name,
					Namespace: proxy.Namespace,
				},
			})
			log.Info("Ingress changed, requeuing TrafiksProxy",
				"ingress", ingress.Name,
				"trafiksProxy", proxy.Name)
		}
	}

	return requests
}

func (r *TrafiksProxyReconciler) shouldReconcileProxyForIngress(ingress *networkingv1.Ingress, proxy *proxyv1.TrafiksProxy) bool {
	if proxy.Status.IngressRef != nil {
		return proxy.Status.IngressRef.Name == ingress.Name &&
			proxy.Status.IngressRef.Namespace == ingress.Namespace
	}

	proxyURL := ingress.Annotations["trafiks.io/proxy-url"]
	return proxyURL != "" && proxy.Spec.ProxyURL == proxyURL
}

func (r *TrafiksProxyReconciler) deleteProjectFromBackend(
	ctx context.Context,
	proxy *proxyv1.TrafiksProxy,
	log logr.Logger,
) error {
	if err := r.getConfiguredAPIClient(ctx, proxy); err != nil {
		return fmt.Errorf("unable to get TrafiksBackend credentials: %w", err)
	}

	if err := r.APIClient.DeleteProject(ctx, proxy.Status.ProjectUID); err != nil {
		return fmt.Errorf("API deletion failed: %w", err)
	}

	log.Info("Successfully deleted project from Trafiks backend",
		"projectUID", proxy.Status.ProjectUID)
	return nil
}

// handleDeletion handles the deletion of a TrafiksProxy resource
func (r *TrafiksProxyReconciler) handleDeletion(ctx context.Context, proxy *proxyv1.TrafiksProxy, log logr.Logger) (ctrl.Result, error) {
	if !containsString(proxy.Finalizers, trafiksProxyFinalizer) {
		return ctrl.Result{}, nil
	}

	if proxy.Status.ProjectUID != "" {
		if err := r.deleteProjectFromBackend(ctx, proxy, log); err != nil {
			log.Error(err, "failed to delete project from Trafiks backend",
				"projectUID", proxy.Status.ProjectUID)
		}
	}

	proxy.Finalizers = removeString(proxy.Finalizers, trafiksProxyFinalizer)
	if err := r.Update(ctx, proxy); err != nil {
		log.Error(err, "unable to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("TrafiksProxy deleted successfully")
	return ctrl.Result{}, nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func (r *TrafiksProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&proxyv1.TrafiksProxy{}).
		Named("trafiksproxy").
		Watches(
			&networkingv1.Ingress{},
			handler.EnqueueRequestsFromMapFunc(r.ingressToTrafiksProxy),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.secretToTrafiksProxy),
		).
		Complete(r)
}

// secretToTrafiksProxy watches TLS secrets and requeues related TrafiksProxy resources
func (r *TrafiksProxyReconciler) secretToTrafiksProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return []reconcile.Request{}
	}

	if secret.Type != corev1.SecretTypeTLS {
		return []reconcile.Request{}
	}

	log := r.Logger
	proxyList := &proxyv1.TrafiksProxyList{}
	if err := r.List(ctx, proxyList); err != nil {
		log.Error(err, "unable to list TrafiksProxy resources")
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for _, proxy := range proxyList.Items {
		if proxy.Spec.Scheme != "https" {
			continue
		}

		if proxy.Spec.TLSCertResolver != "letsencrypt" {
			continue
		}

		if proxy.Status.IngressRef == nil {
			continue
		}

		if proxy.Status.IngressRef.Namespace != secret.Namespace {
			continue
		}

		ingress := &networkingv1.Ingress{}
		ingressKey := types.NamespacedName{
			Name:      proxy.Status.IngressRef.Name,
			Namespace: proxy.Status.IngressRef.Namespace,
		}

		if err := r.Get(ctx, ingressKey, ingress); err != nil {
			continue
		}

		for _, tls := range ingress.Spec.TLS {
			if tls.SecretName != secret.Name {
				continue
			}

			for _, host := range tls.Hosts {
				if host == proxy.Spec.ProxyURL {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      proxy.Name,
							Namespace: proxy.Namespace,
						},
					})
					log.Info("TLS secret updated, requeuing TrafiksProxy",
						"secret", secret.Name,
						"trafiksProxy", proxy.Name,
						"proxyURL", proxy.Spec.ProxyURL)
					break
				}
			}
		}
	}

	return requests
}
