package services

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/matryer/is"
	"go.uber.org/mock/gomock"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/tests/mocks"
)

func TestAPIKey_GenerateAPIKey(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	restErr := pkg.NewRestErr()

	user := &models.User{
		ID: 1,
	}

	payload := dto.GenerateAPIKeyRequest{}

	tests := []struct {
		Name    string
		Data    dto.GenerateAPIKeyRequest
		Prepare func(
			apiKeyRepo *mocks.MockAPIKeyRepoClient,
			userRepo *mocks.MockUserRepoClient,
			webhookRepo *mocks.MockWebhookRepoClient,
			webhookService *mocks.MockWebhookEventSender,
		)
		Expected     func() *pkg.RestErr
		ExpectedResp func() interface{}
		Ctx          func() *gin.Context
	}{
		{
			Name: "Unauthorized - no user in context",
			Data: payload,
			Prepare: func(
				_ *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.Unauthorized("unauthorized")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("POST", "/", nil)
				return ctx
			},
		},
		{
			Name: "Empty name",
			Data: dto.GenerateAPIKeyRequest{Name: "   "},
			Prepare: func(
				_ *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("name is required")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("POST", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Error finding API keys",
			Data: dto.GenerateAPIKeyRequest{Name: "test-key"},
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				apiKeyRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.ServerError("failed to find API keys")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("POST", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Active key already exists",
			Data: dto.GenerateAPIKeyRequest{Name: "test-key"},
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				activeKey := &models.APIKey{
					UID:       "test-uid",
					UserID:    1,
					KeyPrefix: "tfk_xxxx",
					Name:      "existing-key",
					RevokedAt: nil,
					ExpiresAt: nil,
				}
				apiKeyRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any()).
					Return([]*models.APIKey{activeKey}, nil)
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("you already have an active API key. Please revoke it before creating a new one")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("POST", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - create API key",
			Data: dto.GenerateAPIKeyRequest{Name: "test-key"},
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				apiKeyRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any()).
					Return([]*models.APIKey{}, nil)

				apiKeyRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, key *models.APIKey) error {
						key.UID = "test-uid-123"
						key.CreatedAt = time.Now()
						return nil
					})

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("webhook not found"))
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return "not-nil"
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("POST", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - create API key with expiration",
			Data: dto.GenerateAPIKeyRequest{Name: "test-key", ExpiresInDays: 30},
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				apiKeyRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any()).
					Return([]*models.APIKey{}, nil)

				apiKeyRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, key *models.APIKey) error {
						eval.True(key.ExpiresAt != nil)
						key.UID = "test-uid-123"
						key.CreatedAt = time.Now()
						return nil
					})

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("webhook not found"))
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return "not-nil"
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("POST", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - create API key with webhook",
			Data: dto.GenerateAPIKeyRequest{Name: "test-key"},
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				webhookService *mocks.MockWebhookEventSender,
			) {
				apiKeyRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any()).
					Return([]*models.APIKey{}, nil)

				apiKeyRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, key *models.APIKey) error {
						key.UID = "test-uid-123"
						key.CreatedAt = time.Now()
						return nil
					})

				webhook := &models.Webhook{
					ID:       1,
					UserID:   1,
					IsActive: pkg.BoolPtr(true),
				}

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(webhook, nil)

				webhookService.EXPECT().
					SendEvent(gomock.Any(), gomock.Any(), gomock.Any()).
					AnyTimes()
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return "not-nil"
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("POST", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAPIKeyRepo := mocks.NewMockAPIKeyRepoClient(ctrl)
			mockUserRepo := mocks.NewMockUserRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewAPIKey(
				logger,
				mockAPIKeyRepo,
				mockUserRepo,
				mockWebhookRepo,
				restErr,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockAPIKeyRepo,
					mockUserRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.GenerateAPIKey(tt.Ctx(), tt.Data)

			if tt.Expected != nil {
				expectedErr := tt.Expected()
				if expectedErr == nil {
					eval.True(err == nil)
				} else {
					eval.True(err != nil)
					eval.Equal(err.Message, expectedErr.Message)
					eval.Equal(err.StatusCode, expectedErr.StatusCode)
				}
			}

			expectedResp := tt.ExpectedResp()
			if expectedResp == nil {
				eval.True(resp == nil)
			} else if expectedResp == "not-nil" {
				eval.True(resp != nil)
			} else {
				eval.Equal(resp, expectedResp)
			}
		})
	}
}

func TestAPIKey_ListAPIKeys(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	restErr := pkg.NewRestErr()
	logger := pkg.NewLogger()

	user := &models.User{
		ID: 1,
	}

	tests := []struct {
		Name    string
		Prepare func(
			apiKeyRepo *mocks.MockAPIKeyRepoClient,
			userRepo *mocks.MockUserRepoClient,
			webhookRepo *mocks.MockWebhookRepoClient,
			webhookService *mocks.MockWebhookEventSender,
		)
		Expected     func() *pkg.RestErr
		ExpectedResp func() interface{}
		Ctx          func() *gin.Context
	}{
		{
			Name: "Unauthorized - no user in context",
			Prepare: func(
				_ *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.Unauthorized("unauthorized")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("GET", "/", nil)
				return ctx
			},
		},
		{
			Name: "Error fetching API keys",
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				apiKeyRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.ServerError("failed to fetch API keys")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("GET", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - list API keys",
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				apiKeys := []*models.APIKey{
					{
						UID:        "uid-1",
						UserID:     1,
						KeyPrefix:  "tfk_xxxx",
						Name:       "key-1",
						CreatedAt:  now,
						LastUsedAt: &now,
						ExpiresAt:  nil,
						RevokedAt:  nil,
					},
					{
						UID:        "uid-2",
						UserID:     1,
						KeyPrefix:  "tfk_yyyy",
						Name:       "key-2",
						CreatedAt:  now,
						LastUsedAt: nil,
						ExpiresAt:  nil,
						RevokedAt:  &now,
					},
				}

				apiKeyRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any()).
					Return(apiKeys, nil)
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return "not-nil"
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("GET", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAPIKeyRepo := mocks.NewMockAPIKeyRepoClient(ctrl)
			mockUserRepo := mocks.NewMockUserRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewAPIKey(
				logger,
				mockAPIKeyRepo,
				mockUserRepo,
				mockWebhookRepo,
				restErr,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockAPIKeyRepo,
					mockUserRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.ListAPIKeys(tt.Ctx())

			if tt.Expected != nil {
				expectedErr := tt.Expected()
				if expectedErr == nil {
					eval.True(err == nil)
				} else {
					eval.True(err != nil)
					eval.Equal(err.Message, expectedErr.Message)
					eval.Equal(err.StatusCode, expectedErr.StatusCode)
				}
			}

			expectedResp := tt.ExpectedResp()
			if expectedResp == nil {
				eval.True(resp == nil)
			} else if expectedResp == "not-nil" {
				eval.True(resp != nil)
			} else {
				eval.Equal(resp, expectedResp)
			}
		})
	}
}

func TestAPIKey_RevokeAPIKey(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	restErr := pkg.NewRestErr()

	user := &models.User{
		ID: 1,
	}

	tests := []struct {
		Name    string
		KeyID   string
		Prepare func(
			apiKeyRepo *mocks.MockAPIKeyRepoClient,
			userRepo *mocks.MockUserRepoClient,
			webhookRepo *mocks.MockWebhookRepoClient,
			webhookService *mocks.MockWebhookEventSender,
		)
		Expected     func() *pkg.RestErr
		ExpectedResp func() interface{}
		Ctx          func() *gin.Context
	}{
		{
			Name:  "Unauthorized - no user in context",
			KeyID: "test-uid",
			Prepare: func(
				_ *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.Unauthorized("unauthorized")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/", nil)
				return ctx
			},
		},
		{
			Name:  "API key not found",
			KeyID: "test-uid",
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				apiKeyRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.NotFound("API key not found")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name:  "API key already revoked",
			KeyID: "test-uid",
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				apiKey := &models.APIKey{
					UID:       "test-uid",
					UserID:    1,
					RevokedAt: &now,
				}

				apiKeyRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(apiKey, nil)
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("API key is already revoked")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name:  "Success - revoke API key",
			KeyID: "test-uid",
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				apiKey := &models.APIKey{
					UID:       "test-uid",
					UserID:    1,
					KeyPrefix: "tfk_xxxx",
					Name:      "test-key",
					RevokedAt: nil,
				}

				apiKeyRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(apiKey, nil)

				apiKeyRepo.EXPECT().
					Updates(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("webhook not found")).
					AnyTimes()
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return gin.H{
					"message": "API key revoked successfully",
				}
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name:  "Success - revoke API key with webhook",
			KeyID: "test-uid",
			Prepare: func(
				apiKeyRepo *mocks.MockAPIKeyRepoClient,
				_ *mocks.MockUserRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				webhookService *mocks.MockWebhookEventSender,
			) {
				apiKey := &models.APIKey{
					UID:       "test-uid",
					UserID:    1,
					KeyPrefix: "tfk_xxxx",
					Name:      "test-key",
					RevokedAt: nil,
				}

				apiKeyRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(apiKey, nil)

				apiKeyRepo.EXPECT().
					Updates(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				webhook := &models.Webhook{
					ID:       1,
					UserID:   1,
					IsActive: pkg.BoolPtr(true),
				}

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(webhook, nil)

				webhookService.EXPECT().
					SendEvent(gomock.Any(), gomock.Any(), gomock.Any()).
					AnyTimes()
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return gin.H{
					"message": "API key revoked successfully",
				}
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAPIKeyRepo := mocks.NewMockAPIKeyRepoClient(ctrl)
			mockUserRepo := mocks.NewMockUserRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewAPIKey(
				logger,
				mockAPIKeyRepo,
				mockUserRepo,
				mockWebhookRepo,
				restErr,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockAPIKeyRepo,
					mockUserRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.RevokeAPIKey(tt.Ctx(), tt.KeyID)

			if tt.Expected != nil {
				expectedErr := tt.Expected()
				if expectedErr == nil {
					eval.True(err == nil)
				} else {
					eval.True(err != nil)
					eval.Equal(err.Message, expectedErr.Message)
					eval.Equal(err.StatusCode, expectedErr.StatusCode)
				}
			}

			expectedResp := tt.ExpectedResp()
			if expectedResp == nil {
				eval.True(resp == nil)
			} else if expectedResp == "not-nil" {
				eval.True(resp != nil)
			} else {
				eval.Equal(resp, expectedResp)
			}
		})
	}
}
