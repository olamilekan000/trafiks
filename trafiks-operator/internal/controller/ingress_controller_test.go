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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestIngressReconcile_NotFound(t *testing.T) {
	is := is.New(t)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: "default",
		},
	}

	result, err := testIngressReconciler.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, time.Duration(0))
}

func TestIngressReconcile_WithTrafiksClass(t *testing.T) {
	is := is.New(t)

	ns := "default"
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: ns,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPtr("trafiks"),
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

	is.NoErr(k8sClient.Create(context.Background(), ingress))
	defer k8sClient.Delete(context.Background(), ingress)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      ingress.Name,
			Namespace: ingress.Namespace,
		},
	}

	result, err := testIngressReconciler.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, time.Duration(0))
}

func TestIngressReconcile_WithTLS(t *testing.T) {
	is := is.New(t)

	ns := "default"
	certPEM := `-----BEGIN CERTIFICATE-----
MIIDYTCCAkmgAwIBAgIUIzbU1w5lIx9md3kXkspzQjqyObgwDQYJKoZIhvcNAQEL
BQAwQDEeMBwGA1UEAwwVbmdpbngudHJhZmZpa2VkLmNsb3VkMR4wHAYDVQQKDBVu
Z2lueC50cmFmZmlrZWQuY2xvdWQwHhcNMjUxMjE4MDkwMjMxWhcNMjYxMjE4MDkw
MjMxWjBAMR4wHAYDVQQDDBVuZ2lueC50cmFmZmlrZWQuY2xvdWQxHjAcBgNVBAoM
FW5naW54LnRyYWZmaWtlZC5jbG91ZDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBANJGsRJ8dTBVGS/aF8K6h2QvArjHJ/Is8x6s0VVHOmbgwIzB4mdeI/iC
RIdK0YiqgwH1A122dhMs9ZO5B8bkakPNmrAFSZl8fPb2uTpSH9MMxtFM6Hm6EbMF
tLWLMafRs4hExyg0TOlmzENz7xe6SE92vElW72N9Xi9lLRC65QfyuruU/+Y/UtDk
5r1AqzlTzvqO0U9ZmLSd1vPjYmglhYL9OIQ9AsgFxpzcNKHXJM2CDva7GwiGWdQW
MBOipd8TDTPfet68T60qUfxOSpqc9Z6iMytLUQBtablGgFjjsliPDb3k5fo0JP7n
YBzRj8gxs6FYJRXmjJPP5w0seg+MRXkCAwEAAaNTMFEwHQYDVR0OBBYEFIwYcle/
owTblSYNSQBRU/RNJVmfMB8GA1UdIwQYMBaAFIwYcle/owTblSYNSQBRU/RNJVmf
MA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBABVoOU4E72dauOM+
e/4enMVlmWk05Mk4eAyDJX0ikPXVcF4rFKSjF+JYEfLKgl4d80GuYdPzh9OdtYNO
+eCQ06eGAFNvEVp7kT+hD8O28gocQUb9PuAFy2RPz0WLkF/ZN9goMo8P4p0Ami16
CifdH4kytAKUq+E7VGavZ6Cqf3Q8RM248Jw/xvl1ajz+WwmCUUi1RYTDRbwYQATE
9YWxRRwneMZ6ScmVtx5J568+l6xFfntuf8Q52N1cpjZ1Dep0z3nEAYHYKR21YJLH
Lfln/9c6mxHMG7VLaOKzkQargbutzfFLrzp7ROa9WLisfvRmtganrEk0x4akMqVZ
1+lkuko=
-----END CERTIFICATE-----`

	keyPEM := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDSRrESfHUwVRkv
2hfCuodkLwK4xyfyLPMerNFVRzpm4MCMweJnXiP4gkSHStGIqoMB9QNdtnYTLPWT
uQfG5GpDzZqwBUmZfHz29rk6Uh/TDMbRTOh5uhGzBbS1izGn0bOIRMcoNEzpZsxD
c+8XukhPdrxJVu9jfV4vZS0QuuUH8rq7lP/mP1LQ5Oa9QKs5U876jtFPWZi0ndbz
42JoJYWC/TiEPQLIBcac3DSh1yTNgg72uxsIhlnUFjAToqXfEw0z33revE+tKlH8
TkqanPWeojMrS1EAbWm5RoBY47JYjw295OX6NCT+52Ac0Y/IMbOhWCUV5oyTz+cN
LHoPjEV5AgMBAAECggEADm74x5aQarVkqbK0L9Mi6P8LFkjhVo+8Tm45Xjup6Bbk
UKUcV9EP8gZrEshRcqqVnIRHa17TYwSShMXOIVpbavUJeaybPTRc1Yzg8P5jHdcC
LKq1SGm7vkiVe2KnEWhlBOUHsJxKbTj3q2ehUUUhooa5bOVRQEiCNwmgWsYhY3an
fO8upbNTT4VIF0r2nx3JRhJlKhL01V9La/x20sbZkvPTp8zdBI9yV3ejzYlL6O1e
DqMJ6a/8x1qrCpEcYsQ0Ih2240Mm3NMTPT2MuBacklE5Qiia47GIedAKhIbeNy+L
c1wEtHOdxZt3t/UGNlUYKsN+SPqGySnYXONvWuyvYwKBgQDrDdV6tWlOy2MOaMrj
Ai8wLQmnYoDtEisbTH4QQYY0b79pAIoR/gphVK0wCmXpCf0D5bG3ZMDcFvCm5KEJ
MyOti3nTSHkagnkhmTme03GcnCvvNdySmR1hLF5gbMWEPgOfHzQDoWwjhjs0kJ5U
Hx/3kknyQ5bdWm4ogFLMOdeTLwKBgQDlA5zF/K520Dx3z1l5rZLUIjB/+Lu1qPw7
78oAcihb+/jGdXiQP8iGjKxe3ak0lKPqqEOL1Q8wUyFb7bIkkxb1cxVFpl199Aki
8FaLTeE+m8FEBbdMbbgbtWyWX0cGsSLxB5oz5cCRCYSSeETAQ46BTz0H6iNPS45x
p4Xs3kSn1wKBgGr7DOKglqFqKFdykoTnhZqjpPUt/AfqcPwnwGidqftLsQ6VVEIE
Ia1S4NAwq1l5VlLjxBL4JF8HgdgzzqdlQyPFi1kCbzwFjiQgnP0Qt3DUE5r4JMAE
OD719q5kUzFxGCzgAsh0O8efXGr8N1OKJv6C8mz1HkD425JLdWPGH7u5AoGBANBN
etlyvdWADqADT6UnRbgB6Q9dVI8lR1fVAW1qaF9STrNkweaivWf0qAwZngAfewDD
T7zubEROOLd++lveFjHnHWAetEcOIwlOhclrawchcKbIdDLmUWGSoVQdEWN61wdZ
HN87iO3jNFxtXEtspz/irOZ4BunnFYc3Es+iU9w9AoGAKIYlGVpD0eqrr2lEYxK9
KEOkxxYuPy+1A224hevxZWanIf5T0WMZsrWrpUURa8R8e76/MT/u8c1z39Buwuei
z9yN7yK9LGOg1CI4j6oSg5WuU5QPoa8naIC7jq0XMPyh1+I6DWChSlQ4IEhGNq5t
J8nyaOaHFvZ6FFzaAxkE35Y=
-----END PRIVATE KEY-----`

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tls-secret",
			Namespace: ns,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte(certPEM),
			"tls.key": []byte(keyPEM),
		},
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress-tls",
			Namespace: ns,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPtr("trafiks"),
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"test.example.com"},
					SecretName: secret.Name,
				},
			},
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

	is.NoErr(k8sClient.Create(context.Background(), secret))
	defer k8sClient.Delete(context.Background(), secret)

	is.NoErr(k8sClient.Create(context.Background(), ingress))
	defer k8sClient.Delete(context.Background(), ingress)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      ingress.Name,
			Namespace: ingress.Namespace,
		},
	}

	result, err := testIngressReconciler.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, time.Duration(0))
}

func TestIngressReconcile_NonTrafiksClass(t *testing.T) {
	is := is.New(t)

	ns := "default"
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress-other",
			Namespace: ns,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPtr("traefik"),
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
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

	is.NoErr(k8sClient.Create(context.Background(), ingress))
	defer k8sClient.Delete(context.Background(), ingress)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      ingress.Name,
			Namespace: ingress.Namespace,
		},
	}

	result, err := testIngressReconciler.Reconcile(context.Background(), req)
	is.NoErr(err)
	is.Equal(result.RequeueAfter, time.Duration(0))
}

func stringPtr(s string) *string {
	return &s
}

func pathTypePtr(pt networkingv1.PathType) *networkingv1.PathType {
	return &pt
}
