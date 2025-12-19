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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	proxyv1 "github.com/trafiks/operator/api/v1"
	"github.com/trafiks/operator/internal/test/mocks"
)

func TestTrafiksProxyReconcile_NotFound(t *testing.T) {
	is := is.New(t)

	r := &TrafiksProxyReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
		Logger: testLogger,
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

func TestTrafiksProxyReconcile_AddsFinalizer(t *testing.T) {
	is := is.New(t)

	proxy := &proxyv1.TrafiksProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-proxy-finalizer",
			Namespace: "default",
		},
		Spec: proxyv1.TrafiksProxySpec{
			ProxyURL:    "test.example.com",
			ProjectName: "test-project",
			Kubernetes: proxyv1.KubernetesProxyConfig{
				Namespace:       "default",
				ServiceName:     "test-service",
				ServicePortName: "http",
			},
		},
	}

	is.NoErr(k8sClient.Create(context.Background(), proxy))
	defer k8sClient.Delete(context.Background(), proxy)

	r := &TrafiksProxyReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
		Logger: testLogger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      proxy.Name,
			Namespace: proxy.Namespace,
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, time.Duration(0))

	updated := &proxyv1.TrafiksProxy{}
	is.NoErr(k8sClient.Get(context.Background(), req.NamespacedName, updated))
	is.True(containsString(updated.Finalizers, trafiksProxyFinalizer))
}

func TestTrafiksProxyReconcile_WithIngress(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPIClient := mocks.NewMockTrafiksAPIClient(ctrl)

	ns := "default"
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: ns,
			Annotations: map[string]string{
				"trafiks.io/proxy-url":    "test.example.com",
				"trafiks.io/project-name": "test-project",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "test.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pathTypePtr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "test-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}

	is.NoErr(k8sClient.Create(context.Background(), ingress))
	defer k8sClient.Delete(context.Background(), ingress)

	is.NoErr(k8sClient.Create(context.Background(), svc))
	defer k8sClient.Delete(context.Background(), svc)

	proxy := &proxyv1.TrafiksProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-proxy",
			Namespace:  ns,
			Finalizers: []string{trafiksProxyFinalizer},
		},
		Spec: proxyv1.TrafiksProxySpec{
			ProxyURL:    "test.example.com",
			ProjectName: "test-project",
			Kubernetes: proxyv1.KubernetesProxyConfig{
				Namespace:       ns,
				ServiceName:     "test-service",
				ServicePortName: "http",
			},
		},
	}

	is.NoErr(k8sClient.Create(context.Background(), proxy))
	defer k8sClient.Delete(context.Background(), proxy)

	mockAPIClient.EXPECT().SetAPIConfig(gomock.Any(), gomock.Any()).AnyTimes()
	mockAPIClient.EXPECT().ListProjects(gomock.Any()).Return([]map[string]any{}, nil).AnyTimes()
	mockAPIClient.EXPECT().CreateProject(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[string]any{"uid": "project-uid", "name": "test-project"}, nil).AnyTimes()
	mockAPIClient.EXPECT().GetService(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	mockAPIClient.EXPECT().CreateService(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[string]any{"uid": "service-uid"}, nil).AnyTimes()

	r := &TrafiksProxyReconciler{
		Client:    k8sClient,
		Scheme:    k8sClient.Scheme(),
		Logger:    testLogger,
		APIClient: mockAPIClient,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      proxy.Name,
			Namespace: proxy.Namespace,
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	is.NoErr(err)

	updated := &proxyv1.TrafiksProxy{}
	is.NoErr(k8sClient.Get(context.Background(), req.NamespacedName, updated))
	is.True(updated.Status.IngressRef != nil)
	is.Equal(updated.Status.IngressRef.Name, ingress.Name)
}
