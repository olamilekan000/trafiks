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
	"testing"
	"time"

	"github.com/matryer/is"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	proxyv1 "github.com/trafiks/operator/api/v1"
	"github.com/trafiks/operator/internal/test/mocks"
)

func TestTrafiksBackendReconcile_NotFound(t *testing.T) {
	is := is.New(t)

	r := &TrafiksBackendReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, time.Duration(0))
}

func TestTrafiksBackendReconcile_SecretNotFound(t *testing.T) {
	is := is.New(t)

	ns := "default"
	backend := &proxyv1.TrafiksBackend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backend",
			Namespace: ns,
		},
		Spec: proxyv1.TrafiksBackendSpec{
			SecretRef: proxyv1.SecretReference{
				Name:      "non-existent-secret",
				Namespace: ns,
			},
		},
	}

	is.NoErr(k8sClient.Create(context.Background(), backend))
	defer k8sClient.Delete(context.Background(), backend)

	r := &TrafiksBackendReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      backend.Name,
			Namespace: backend.Namespace,
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, 30*time.Second)

	updated := &proxyv1.TrafiksBackend{}
	is.NoErr(k8sClient.Get(context.Background(), req.NamespacedName, updated))

	readyCond := findCondition(updated.Status.Conditions, ConditionTypeReady)
	is.True(readyCond != nil)
	is.Equal(readyCond.Status, metav1.ConditionFalse)
	is.Equal(readyCond.Reason, "SecretNotFound")

	availableCond := findCondition(updated.Status.Conditions, ConditionTypeAvailable)
	is.True(availableCond != nil)
	is.Equal(availableCond.Status, metav1.ConditionUnknown)
	is.Equal(availableCond.Reason, "SecretNotFound")
}

func TestTrafiksBackendReconcile_MissingSecretKeys(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ns := "default"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: ns,
		},
		Data: map[string][]byte{},
	}

	backend := &proxyv1.TrafiksBackend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backend",
			Namespace: ns,
		},
		Spec: proxyv1.TrafiksBackendSpec{
			SecretRef: proxyv1.SecretReference{
				Name:      secret.Name,
				Namespace: ns,
			},
		},
	}

	is.NoErr(k8sClient.Create(context.Background(), secret))
	defer k8sClient.Delete(context.Background(), secret)

	is.NoErr(k8sClient.Create(context.Background(), backend))
	defer k8sClient.Delete(context.Background(), backend)

	r := &TrafiksBackendReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      backend.Name,
			Namespace: backend.Namespace,
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, 30*time.Second)

	updated := &proxyv1.TrafiksBackend{}
	is.NoErr(k8sClient.Get(context.Background(), req.NamespacedName, updated))

	readyCond := findCondition(updated.Status.Conditions, ConditionTypeReady)
	is.True(readyCond != nil)
	is.Equal(readyCond.Status, metav1.ConditionFalse)
	is.Equal(readyCond.Reason, "MissingSecretKeys")

	availableCond := findCondition(updated.Status.Conditions, ConditionTypeAvailable)
	is.True(availableCond != nil)
	is.Equal(availableCond.Status, metav1.ConditionUnknown)
	is.Equal(availableCond.Reason, "MissingSecretKeys")
}

func TestTrafiksBackendReconcile_EmptySecretValues(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ns := "default"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret-empty",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"baseURL": []byte(""),
			"apiKey":  []byte(""),
		},
	}

	backend := &proxyv1.TrafiksBackend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backend-empty",
			Namespace: ns,
		},
		Spec: proxyv1.TrafiksBackendSpec{
			SecretRef: proxyv1.SecretReference{
				Name:      secret.Name,
				Namespace: ns,
			},
		},
	}

	is.NoErr(k8sClient.Create(context.Background(), secret))
	defer k8sClient.Delete(context.Background(), secret)

	is.NoErr(k8sClient.Create(context.Background(), backend))
	defer k8sClient.Delete(context.Background(), backend)

	r := &TrafiksBackendReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      backend.Name,
			Namespace: backend.Namespace,
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, 30*time.Second)

	updated := &proxyv1.TrafiksBackend{}
	is.NoErr(k8sClient.Get(context.Background(), req.NamespacedName, updated))

	readyCond := findCondition(updated.Status.Conditions, ConditionTypeReady)
	is.True(readyCond != nil)
	is.Equal(readyCond.Status, metav1.ConditionFalse)
	is.Equal(readyCond.Reason, "EmptySecretValues")

	availableCond := findCondition(updated.Status.Conditions, ConditionTypeAvailable)
	is.True(availableCond != nil)
	is.Equal(availableCond.Status, metav1.ConditionUnknown)
	is.Equal(availableCond.Reason, "EmptySecretValues")
}

func TestTrafiksBackendReconcile_Success(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIClient := mocks.NewMockTrafiksAPIClient(ctrl)

	ns := "default"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret-success",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"baseURL": []byte("http://localhost:8080"),
			"apiKey":  []byte("test-api-key"),
		},
	}

	backend := &proxyv1.TrafiksBackend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backend-success",
			Namespace: ns,
		},
		Spec: proxyv1.TrafiksBackendSpec{
			SecretRef: proxyv1.SecretReference{
				Name:      secret.Name,
				Namespace: ns,
			},
		},
	}

	is.NoErr(k8sClient.Create(context.Background(), secret))
	defer k8sClient.Delete(context.Background(), secret)

	is.NoErr(k8sClient.Create(context.Background(), backend))
	defer k8sClient.Delete(context.Background(), backend)

	mockAPIClient.EXPECT().SetAPIConfig("http://localhost:8080", "test-api-key").AnyTimes()
	mockAPIClient.EXPECT().CheckHealth(gomock.Any()).Return(200, nil).AnyTimes()
	mockAPIClient.EXPECT().CheckAuthentication(gomock.Any()).Return(200, nil).AnyTimes()

	r := &TrafiksBackendReconciler{
		Client:    k8sClient,
		Scheme:    k8sClient.Scheme(),
		APIClient: mockAPIClient,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      backend.Name,
			Namespace: backend.Namespace,
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, 1*time.Minute) // Controller requeues every minute to check backend health

	updated := &proxyv1.TrafiksBackend{}
	is.NoErr(k8sClient.Get(context.Background(), req.NamespacedName, updated))

	readyCond := findCondition(updated.Status.Conditions, ConditionTypeReady)
	is.True(readyCond != nil)
	is.Equal(readyCond.Status, metav1.ConditionTrue)
	is.Equal(readyCond.Reason, "SecretValidAndBackendReachable")

	availableCond := findCondition(updated.Status.Conditions, ConditionTypeAvailable)
	is.True(availableCond != nil)
	is.Equal(availableCond.Status, metav1.ConditionTrue)
	is.Equal(availableCond.Reason, "BackendReachable")
}
