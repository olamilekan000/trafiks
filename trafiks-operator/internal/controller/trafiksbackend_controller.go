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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	proxyv1 "github.com/trafiks/operator/api/v1"
)

// TrafiksBackendReconciler reconciles a TrafiksBackend object
type TrafiksBackendReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	APIClient TrafiksAPIClient
}

// +kubebuilder:rbac:groups=proxy.trafiks.io,resources=trafiksbackends,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=proxy.trafiks.io,resources=trafiksbackends/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=proxy.trafiks.io,resources=trafiksbackends/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TrafiksBackendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	backend := &proxyv1.TrafiksBackend{}
	if err := r.Get(ctx, req.NamespacedName, backend); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch TrafiksBackend")
		return ctrl.Result{}, err
	}

	if backend.Status.Conditions == nil {
		backend.Status.Conditions = []metav1.Condition{}
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
		if apierrors.IsNotFound(err) {
			SetCondition(&backend.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse, "SecretNotFound",
				fmt.Sprintf("Secret %s not found in namespace %s", secretRef.Name, secretNamespace), backend.Generation)
			SetCondition(&backend.Status.Conditions, ConditionTypeAvailable, metav1.ConditionUnknown, "SecretNotFound",
				"Cannot check backend availability: secret not found", backend.Generation)
			if err := r.updateStatus(ctx, req.NamespacedName, backend.Status.Conditions, backend.Generation); err != nil {
				log.Error(err, "unable to update TrafiksBackend status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		log.Error(err, "unable to fetch secret")
		return ctrl.Result{}, err
	}

	baseURLBytes, baseURLExists := secret.Data[baseURLKey]
	apiKeyBytes, apiKeyExists := secret.Data[apiKeyKey]

	if !baseURLExists || !apiKeyExists {
		missingKeys := []string{}
		if !baseURLExists {
			missingKeys = append(missingKeys, baseURLKey)
		}
		if !apiKeyExists {
			missingKeys = append(missingKeys, apiKeyKey)
		}
		SetCondition(&backend.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse, "MissingSecretKeys",
			fmt.Sprintf("Secret missing required keys: %v", missingKeys), backend.Generation)
		SetCondition(&backend.Status.Conditions, ConditionTypeAvailable, metav1.ConditionUnknown, "MissingSecretKeys",
			"Cannot check backend availability: secret missing required keys", backend.Generation)
		if err := r.updateStatus(ctx, req.NamespacedName, backend.Status.Conditions, backend.Generation); err != nil {
			log.Error(err, "unable to update TrafiksBackend status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	baseURL := string(baseURLBytes)
	apiKey := string(apiKeyBytes)

	if baseURL == "" || apiKey == "" {
		SetCondition(&backend.Status.Conditions, ConditionTypeReady, metav1.ConditionFalse, "EmptySecretValues",
			"Secret contains empty values for baseURL or apiKey", backend.Generation)
		SetCondition(&backend.Status.Conditions, ConditionTypeAvailable, metav1.ConditionUnknown, "EmptySecretValues",
			"Cannot check backend availability: secret contains empty values", backend.Generation)
		if err := r.updateStatus(ctx, req.NamespacedName, backend.Status.Conditions, backend.Generation); err != nil {
			log.Error(err, "unable to update TrafiksBackend status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	available, availableReason, availableMessage := r.checkBackendReachable(ctx, baseURL, apiKey)
	ready, readyReason, readyMessage := r.checkAuthentication(ctx, baseURL, apiKey)

	SetCondition(&backend.Status.Conditions, ConditionTypeAvailable, available, availableReason, availableMessage, backend.Generation)
	SetCondition(&backend.Status.Conditions, ConditionTypeReady, ready, readyReason, readyMessage, backend.Generation)

	if err := r.updateStatus(ctx, req.NamespacedName, backend.Status.Conditions, backend.Generation); err != nil {
		log.Error(err, "unable to update TrafiksBackend status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *TrafiksBackendReconciler) checkBackendReachable(ctx context.Context, baseURL, apiKey string) (metav1.ConditionStatus, string, string) {
	r.APIClient.SetAPIConfig(baseURL, apiKey)
	statusCode, err := r.APIClient.CheckHealth(ctx)
	if err != nil {
		return metav1.ConditionFalse, "BackendUnreachable",
			fmt.Sprintf("Cannot reach Trafiks backend: %v", err)
	}

	if statusCode != 200 {
		return metav1.ConditionFalse, "BackendUnhealthy",
			fmt.Sprintf("Trafiks backend returned status %d", statusCode)
	}

	return metav1.ConditionTrue, "BackendReachable", "Trafiks backend is reachable"
}

func (r *TrafiksBackendReconciler) checkAuthentication(ctx context.Context, baseURL, apiKey string) (metav1.ConditionStatus, string, string) {
	r.APIClient.SetAPIConfig(baseURL, apiKey)
	statusCode, err := r.APIClient.CheckAuthentication(ctx)
	if err != nil {
		return metav1.ConditionFalse, "BackendUnreachable",
			fmt.Sprintf("Cannot reach Trafiks backend for authentication: %v", err)
	}

	if statusCode == 401 {
		return metav1.ConditionFalse, "AuthenticationFailed",
			"API key authentication failed: invalid credentials"
	}

	if statusCode != 200 {
		return metav1.ConditionFalse, "AuthenticationFailed",
			fmt.Sprintf("Authentication check failed with status %d", statusCode)
	}

	return metav1.ConditionTrue, "SecretValidAndBackendReachable",
		"Trafiks backend is ready and accessible"
}

func (r *TrafiksBackendReconciler) updateStatus(ctx context.Context, nn types.NamespacedName, conditions []metav1.Condition, observedGeneration int64) error {
	backend := &proxyv1.TrafiksBackend{}
	return updateStatusWithRetry(ctx, r.Client, nn, backend, func(obj client.Object) {
		backend := obj.(*proxyv1.TrafiksBackend)
		backend.Status.Conditions = conditions
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *TrafiksBackendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&proxyv1.TrafiksBackend{}).
		Named("trafiksbackend").
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.secretToTrafiksBackend),
		).
		Complete(r)
}

func (r *TrafiksBackendReconciler) secretToTrafiksBackend(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return []reconcile.Request{}
	}

	log := log.FromContext(ctx)

	backendList := &proxyv1.TrafiksBackendList{}
	if err := r.List(ctx, backendList); err != nil {
		log.Error(err, "unable to list TrafiksBackend resources")
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for _, backend := range backendList.Items {
		secretRef := backend.Spec.SecretRef
		secretNamespace := secretRef.Namespace
		if secretNamespace == "" {
			secretNamespace = backend.Namespace
		}

		if secret.Name == secretRef.Name && secret.Namespace == secretNamespace {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      backend.Name,
					Namespace: backend.Namespace,
				},
			})

			log.Info("Secret changed, requeuing TrafiksBackend",
				"secret", secret.Name,
				"secretNamespace", secret.Namespace,
				"trafiksBackend", backend.Name,
				"trafiksBackendNamespace", backend.Namespace,
			)
		}
	}

	return requests
}
