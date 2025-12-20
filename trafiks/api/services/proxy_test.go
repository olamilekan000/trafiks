package services

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"

	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/pkg/source"
	"github.com/trafiks/trafiks/tests/mocks"
)

func TestProxyService_ProxyRequest(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	baseURL := "http://localhost:8080"
	tlsPort := "443"
	mockStreamHub := pkg.NewMetricsStreamHub()
	mockWebhookService := mocks.NewMockWebhookEventSender(gomock.NewController(t))

	tests := []struct {
		Name    string
		Prepare func(
			serviceRepo *mocks.MockServiceRepoClient,
			projectRepo *mocks.MockProjectRepoClient,
			requestLogRepo *mocks.MockProxyRequestLogRepoClient,
			webhookRepo *mocks.MockWebhookRepoClient,
			cacheClient *mocks.MockCache,
			sourceManager *source.ServiceSourceManager,
			mockSource *mocks.MockServiceSource,
			httpClient *MockHTTPClient,
		)
		Expected  func() *ProxyResponse
		Req       func() *http.Request
		ClientIP  string
		UserAgent string
	}{
		{
			Name: "Invalid proxy URL - empty host",
			Prepare: func(
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				_ *source.ServiceSourceManager,
				_ *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				// No mocks needed
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusBadRequest,
					ErrorMessage:   "Invalid proxy URL",
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Host = ""
				return req
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "Service not found in repository",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				_ *source.ServiceSourceManager,
				_ *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, gorm.ErrRecordNotFound)
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusNotFound,
					ErrorMessage:   "Service not found",
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "Error finding service in repository",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				_ *source.ServiceSourceManager,
				_ *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusInternalServerError,
					ErrorMessage:   "Failed to find service",
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "Service not found in source manager",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				sourceManager *source.ServiceSourceManager,
				mockSource *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				service := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
				}

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				mockSource.EXPECT().Name().Return("trafiks").AnyTimes()
				mockSource.EXPECT().Enabled().Return(true).AnyTimes()
				mockSource.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, gorm.ErrRecordNotFound)

				sourceManager.Register(mockSource)
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusNotFound,
					ErrorMessage:   "Service not found",
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "Project is inactive",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				requestLogRepo *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				sourceManager *source.ServiceSourceManager,
				mockSource *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				service := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
					Project: models.Project{
						ID: 1,
					},
				}

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				isActive := false
				serviceWithProject := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
					Project: models.Project{
						ID:       1,
						IsActive: &isActive,
						Slug:     "test-project",
					},
				}

				mockSource.EXPECT().Name().Return("trafiks").AnyTimes()
				mockSource.EXPECT().Enabled().Return(true).AnyTimes()
				mockSource.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(serviceWithProject, nil)

				sourceManager.Register(mockSource)

				requestLogRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusForbidden,
					ErrorMessage:   "Project is inactive",
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "Invalid service configuration",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				requestLogRepo *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				sourceManager *source.ServiceSourceManager,
				mockSource *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				service := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
					Project: models.Project{
						ID: 1,
					},
				}

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				isActive := true
				serviceWithProject := &models.Service{
					ProxyURL:      "test.example.com",
					Source:        "trafiks",
					Configuration: []byte(`invalid json`),
					Project: models.Project{
						ID:       1,
						IsActive: &isActive,
					},
				}

				mockSource.EXPECT().Name().Return("trafiks").AnyTimes()
				mockSource.EXPECT().Enabled().Return(true).AnyTimes()
				mockSource.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(serviceWithProject, nil)

				sourceManager.Register(mockSource)

				requestLogRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusInternalServerError,
					ErrorMessage:   "failed to parse service config",
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "HTTPS service - HTTP request with redirect enabled",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				requestLogRepo *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				sourceManager *source.ServiceSourceManager,
				mockSource *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				service := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
					Scheme:   models.SchemeHTTPS,
					Project: models.Project{
						ID: 1,
					},
				}

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				isActive := true
				httpsRedirect := true
				config := &models.ServiceConfig{
					HTTPSRedirect: &httpsRedirect,
				}

				serviceWithProject := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
					Scheme:   models.SchemeHTTPS,
					Project: models.Project{
						ID:       1,
						IsActive: &isActive,
					},
				}
				serviceWithProject.SetConfig(config)

				mockSource.EXPECT().Name().Return("trafiks").AnyTimes()
				mockSource.EXPECT().Enabled().Return(true).AnyTimes()
				mockSource.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(serviceWithProject, nil)

				sourceManager.Register(mockSource)

				requestLogRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusMovedPermanently,
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "HTTPS service - HTTP request without redirect",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				requestLogRepo *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockCache,
				sourceManager *source.ServiceSourceManager,
				mockSource *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				service := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
					Scheme:   models.SchemeHTTPS,
					Project: models.Project{
						ID: 1,
					},
				}

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				isActive := true
				httpsRedirect := false
				config := &models.ServiceConfig{
					HTTPSRedirect: &httpsRedirect,
				}

				serviceWithProject := &models.Service{
					ProxyURL: "test.example.com",
					Source:   "trafiks",
					Scheme:   models.SchemeHTTPS,
					Project: models.Project{
						ID:       1,
						IsActive: &isActive,
					},
				}
				serviceWithProject.SetConfig(config)

				mockSource.EXPECT().Name().Return("trafiks").AnyTimes()
				mockSource.EXPECT().Enabled().Return(true).AnyTimes()
				mockSource.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(serviceWithProject, nil)

				sourceManager.Register(mockSource)

				requestLogRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusForbidden,
					ErrorMessage:   "HTTPS required for this service",
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "Cache hit",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				requestLogRepo *mocks.MockProxyRequestLogRepoClient,
				_ *mocks.MockWebhookRepoClient,
				cacheClient *mocks.MockCache,
				sourceManager *source.ServiceSourceManager,
				mockSource *mocks.MockServiceSource,
				_ *MockHTTPClient,
			) {
				service := &models.Service{
					ProxyURL:     "test.example.com",
					Source:       "trafiks",
					Scheme:       models.SchemeHTTP,
					CacheEnabled: true,
					Project: models.Project{
						ID: 1,
					},
				}

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				isActive := true
				serviceWithProject := &models.Service{
					ProxyURL:     "test.example.com",
					Source:       "trafiks",
					Scheme:       models.SchemeHTTP,
					CacheEnabled: true,
					Project: models.Project{
						ID:       1,
						IsActive: &isActive,
					},
				}

				mockSource.EXPECT().Name().Return("trafiks").AnyTimes()
				mockSource.EXPECT().Enabled().Return(true).AnyTimes()
				mockSource.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(serviceWithProject, nil)

				sourceManager.Register(mockSource)

				cacheClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return([]byte(`{"cached": true}`), nil)

				requestLogRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusOK,
					CacheHit:       true,
					Body:           []byte(`{"cached": true}`),
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
		{
			Name: "Success - proxy request",
			Prepare: func(
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockProjectRepoClient,
				requestLogRepo *mocks.MockProxyRequestLogRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				cacheClient *mocks.MockCache,
				sourceManager *source.ServiceSourceManager,
				mockSource *mocks.MockServiceSource,
				httpClient *MockHTTPClient,
			) {
				service := &models.Service{
					ProxyURL:         "test.example.com",
					Source:           "trafiks",
					Scheme:           models.SchemeHTTP,
					TargetBackendURL: "http://backend:8080",
					Project: models.Project{
						ID:     1,
						UserID: 1,
					},
				}

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				isActive := true
				serviceWithProject := &models.Service{
					UID:              "service-uid",
					ProxyURL:         "test.example.com",
					Source:           "trafiks",
					Scheme:           models.SchemeHTTP,
					TargetBackendURL: "http://backend:8080",
					Project: models.Project{
						ID:       1,
						UserID:   1,
						IsActive: &isActive,
						UID:      "project-uid",
					},
				}

				mockSource.EXPECT().Name().Return("trafiks").AnyTimes()
				mockSource.EXPECT().Enabled().Return(true).AnyTimes()
				mockSource.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(serviceWithProject, nil)

				sourceManager.Register(mockSource)

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("webhook not found")).
					AnyTimes()

				httpClient.DoFunc = func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"success": true}`))),
						Header:     make(http.Header),
					}
					resp.Header.Set("Content-Type", "application/json")
					return resp, nil
				}

				requestLogRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			Expected: func() *ProxyResponse {
				return &ProxyResponse{
					StatusCode:     http.StatusOK,
					Body:           []byte(`{"success": true}`),
					Headers:        make(http.Header),
					RequestHeaders: make(map[string]string),
				}
			},
			Req: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			ClientIP:  "127.0.0.1",
			UserAgent: "test-agent",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockServiceRepo := mocks.NewMockServiceRepoClient(ctrl)
			mockProjectRepo := mocks.NewMockProjectRepoClient(ctrl)
			mockRequestLogRepo := mocks.NewMockProxyRequestLogRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockCache := mocks.NewMockCache(ctrl)
			sourceManager := source.NewServiceSourceManager()
			mockSource := mocks.NewMockServiceSource(ctrl)
			mockHTTPClient := &MockHTTPClient{}

			proxyService := NewProxyService(
				logger,
				mockServiceRepo,
				mockProjectRepo,
				mockRequestLogRepo,
				mockCache,
				baseURL,
				tlsPort,
				mockStreamHub,
				mockWebhookService,
				sourceManager,
				mockWebhookRepo,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockServiceRepo,
					mockProjectRepo,
					mockRequestLogRepo,
					mockWebhookRepo,
					mockCache,
					sourceManager,
					mockSource,
					mockHTTPClient,
				)
			}

			proxyService.transport = &mockHTTPTransport{doFunc: mockHTTPClient.DoFunc}

			req := tt.Req()
			resp := proxyService.ProxyRequest(req, tt.ClientIP, tt.UserAgent)

			expected := tt.Expected()
			if expected != nil {
				eval.Equal(resp.StatusCode, expected.StatusCode)
				if expected.ErrorMessage != "" {
					eval.Equal(resp.ErrorMessage, expected.ErrorMessage)
				}
				if expected.CacheHit {
					eval.True(resp.CacheHit)
				}
				if len(expected.Body) > 0 {
					eval.Equal(string(resp.Body), string(expected.Body))
				}
			}
		})
	}
}

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("not implemented")
}

type mockHTTPTransport struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return nil, errors.New("not implemented")
}
