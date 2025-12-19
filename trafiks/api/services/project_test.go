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
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/tests/mocks"
)

func TestProject_CreateProject(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	restErr := pkg.NewRestErr()
	config := &cfg.Config{}

	user := &models.User{
		ID: 1,
	}

	payload := dto.CreateProjectRequest{}

	tests := []struct {
		Name    string
		Data    dto.CreateProjectRequest
		Prepare func(
			projectRepo *mocks.MockProjectRepoClient,
			serviceRepo *mocks.MockServiceRepoClient,
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
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
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
			Name: "Invalid request - empty name",
			Data: dto.CreateProjectRequest{Name: "   "},
			Prepare: func(
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
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
			Name: "Invalid request - name too long",
			Data: dto.CreateProjectRequest{Name: string(make([]byte, 101))},
			Prepare: func(
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("name must not exceed 100 characters")
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
			Name: "Error creating project",
			Data: dto.CreateProjectRequest{Name: "test-project"},
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				projectRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.ServerError("failed to create project")
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
			Name: "Success - create project",
			Data: dto.CreateProjectRequest{Name: "test-project", Description: "test description"},
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				projectRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, project *models.Project) error {
						project.UID = "test-uid-123"
						project.Slug = "test-project"
						project.CreatedAt = time.Now()
						project.UpdatedAt = time.Now()
						isActive := true
						project.IsActive = &isActive
						return nil
					})
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

			mockProjectRepo := mocks.NewMockProjectRepoClient(ctrl)
			mockServiceRepo := mocks.NewMockServiceRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewProject(
				logger,
				mockProjectRepo,
				mockServiceRepo,
				mockWebhookRepo,
				restErr,
				config,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockProjectRepo,
					mockServiceRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.CreateProject(tt.Ctx(), tt.Data)

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

func TestProject_ListProjects(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	restErr := pkg.NewRestErr()
	config := &cfg.Config{}

	user := &models.User{
		ID: 1,
	}

	tests := []struct {
		Name    string
		Prepare func(
			projectRepo *mocks.MockProjectRepoClient,
			serviceRepo *mocks.MockServiceRepoClient,
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
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
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
			Name: "Error fetching projects",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				projectRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.ServerError("failed to fetch projects")
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
			Name: "Success - list projects",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := true
				projects := []*models.Project{
					{
						UID:         "uid-1",
						UserID:      1,
						Name:        "project-1",
						Slug:        "project-1",
						Description: "description-1",
						IsActive:    &isActive,
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				}

				projectRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, filter *models.Project, pagination *dto.Pagination) ([]*models.Project, error) {
						pagination.Total = 1
						return projects, nil
					})
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
		{
			Name: "Success - list projects with pagination",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := true
				projects := []*models.Project{
					{
						UID:         "uid-1",
						UserID:      1,
						Name:        "project-1",
						Slug:        "project-1",
						Description: "description-1",
						IsActive:    &isActive,
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				}

				projectRepo.EXPECT().
					FindMany(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, filter *models.Project, pagination *dto.Pagination) ([]*models.Project, error) {
						pagination.Total = 1
						return projects, nil
					})
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
				ctx.Request = httptest.NewRequest("GET", "/?page=2&limit=10", nil)
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

			mockProjectRepo := mocks.NewMockProjectRepoClient(ctrl)
			mockServiceRepo := mocks.NewMockServiceRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewProject(
				logger,
				mockProjectRepo,
				mockServiceRepo,
				mockWebhookRepo,
				restErr,
				config,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockProjectRepo,
					mockServiceRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.ListProjects(tt.Ctx())

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

func TestProject_GetProject(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	restErr := pkg.NewRestErr()
	config := &cfg.Config{}

	user := &models.User{
		ID: 1,
	}

	tests := []struct {
		Name    string
		Prepare func(
			projectRepo *mocks.MockProjectRepoClient,
			serviceRepo *mocks.MockServiceRepoClient,
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
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
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
				ctx.Request = httptest.NewRequest("GET", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				return ctx
			},
		},
		{
			Name: "Missing project ID",
			Prepare: func(
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("project ID is required")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("GET", "/projects/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Project not found",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.NotFound("project not found")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("GET", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - get project",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := true
				project := &models.Project{
					UID:         "test-uid",
					UserID:      1,
					Name:        "test-project",
					Slug:        "test-project",
					Description: "test description",
					IsActive:    &isActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(project, nil)
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
				ctx.Request = httptest.NewRequest("GET", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
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

			mockProjectRepo := mocks.NewMockProjectRepoClient(ctrl)
			mockServiceRepo := mocks.NewMockServiceRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewProject(
				logger,
				mockProjectRepo,
				mockServiceRepo,
				mockWebhookRepo,
				restErr,
				config,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockProjectRepo,
					mockServiceRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.GetProject(tt.Ctx())

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

func TestProject_UpdateProject(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	restErr := pkg.NewRestErr()
	config := &cfg.Config{}

	user := &models.User{
		ID: 1,
	}

	tests := []struct {
		Name    string
		Data    dto.UpdateProjectRequest
		Prepare func(
			projectRepo *mocks.MockProjectRepoClient,
			serviceRepo *mocks.MockServiceRepoClient,
			webhookRepo *mocks.MockWebhookRepoClient,
			webhookService *mocks.MockWebhookEventSender,
		)
		Expected     func() *pkg.RestErr
		ExpectedResp func() interface{}
		Ctx          func() *gin.Context
	}{
		{
			Name: "Unauthorized - no user in context",
			Data: dto.UpdateProjectRequest{},
			Prepare: func(
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
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
				ctx.Request = httptest.NewRequest("PUT", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				return ctx
			},
		},
		{
			Name: "Missing project ID",
			Data: dto.UpdateProjectRequest{Description: "new description"},
			Prepare: func(
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("project ID is required")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("PUT", "/projects/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Project not found",
			Data: dto.UpdateProjectRequest{Description: "new description"},
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.NotFound("project not found")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("PUT", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "No fields to update",
			Data: dto.UpdateProjectRequest{},
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := true
				project := &models.Project{
					ID:          1,
					UID:         "test-uid",
					UserID:      1,
					Name:        "test-project",
					Slug:        "test-project",
					Description: "test description",
					IsActive:    &isActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(project, nil)
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("no fields to update")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("PUT", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - update description",
			Data: dto.UpdateProjectRequest{Description: "new description"},
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := true
				project := &models.Project{
					ID:          1,
					UID:         "test-uid",
					UserID:      1,
					Name:        "test-project",
					Slug:        "test-project",
					Description: "test description",
					IsActive:    &isActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(project, nil)

				projectRepo.EXPECT().
					Updates(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
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
				ctx.Request = httptest.NewRequest("PUT", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "IsActive set but no webhook found - returns nil",
			Data: dto.UpdateProjectRequest{IsActive: pkg.BoolPtr(true)},
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := false
				project := &models.Project{
					ID:          1,
					UID:         "test-uid",
					UserID:      1,
					Name:        "test-project",
					Slug:        "test-project",
					Description: "test description",
					IsActive:    &isActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(project, nil)

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("webhook not found"))
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("PUT", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - activate project",
			Data: dto.UpdateProjectRequest{IsActive: pkg.BoolPtr(true)},
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				webhookRepo *mocks.MockWebhookRepoClient,
				webhookService *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := false
				project := &models.Project{
					ID:          1,
					UID:         "test-uid",
					UserID:      1,
					Name:        "test-project",
					Slug:        "test-project",
					Description: "test description",
					IsActive:    &isActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(project, nil)

				webhook := &models.Webhook{
					ID:       1,
					UserID:   1,
					IsActive: pkg.BoolPtr(true),
				}

				webhookRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(webhook, nil)

				projectRepo.EXPECT().
					Updates(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

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
				ctx.Request = httptest.NewRequest("PUT", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
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

			mockProjectRepo := mocks.NewMockProjectRepoClient(ctrl)
			mockServiceRepo := mocks.NewMockServiceRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewProject(
				logger,
				mockProjectRepo,
				mockServiceRepo,
				mockWebhookRepo,
				restErr,
				config,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockProjectRepo,
					mockServiceRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.UpdateProject(tt.Ctx(), tt.Data)

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

func TestProject_DeleteProject(t *testing.T) {
	t.Parallel()

	eval := is.New(t)

	logger := pkg.NewLogger()
	restErr := pkg.NewRestErr()
	config := &cfg.Config{}

	user := &models.User{
		ID: 1,
	}

	tests := []struct {
		Name    string
		Prepare func(
			projectRepo *mocks.MockProjectRepoClient,
			serviceRepo *mocks.MockServiceRepoClient,
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
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
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
				ctx.Request = httptest.NewRequest("DELETE", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				return ctx
			},
		},
		{
			Name: "Missing project ID",
			Prepare: func(
				_ *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				// No mocks needed
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.BadRequest("project ID is required")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/projects/", nil)
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Project not found",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				_ *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			Expected: func() *pkg.RestErr {
				re := pkg.RestErr{}
				return re.NotFound("project not found")
			},
			ExpectedResp: func() interface{} {
				return nil
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - delete project without service",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := true
				project := &models.Project{
					ID:          1,
					UID:         "test-uid",
					UserID:      1,
					Name:        "test-project",
					Slug:        "test-project",
					Description: "test description",
					IsActive:    &isActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(project, nil)

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("service not found"))

				projectRepo.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return gin.H{
					"message": "project deleted successfully",
				}
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
				ctx.Set(AppUserContext, user)
				return ctx
			},
		},
		{
			Name: "Success - delete project with service",
			Prepare: func(
				projectRepo *mocks.MockProjectRepoClient,
				serviceRepo *mocks.MockServiceRepoClient,
				_ *mocks.MockWebhookRepoClient,
				_ *mocks.MockWebhookEventSender,
			) {
				now := time.Now()
				isActive := true
				project := &models.Project{
					ID:          1,
					UID:         "test-uid",
					UserID:      1,
					Name:        "test-project",
					Slug:        "test-project",
					Description: "test description",
					IsActive:    &isActive,
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				service := &models.Service{
					ID:        1,
					ProjectID: 1,
				}

				projectRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(project, nil)

				serviceRepo.EXPECT().
					Find(gomock.Any(), gomock.Any()).
					Return(service, nil)

				serviceRepo.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil)

				projectRepo.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			Expected: func() *pkg.RestErr {
				return nil
			},
			ExpectedResp: func() interface{} {
				return gin.H{
					"message": "project deleted successfully",
				}
			},
			Ctx: func() *gin.Context {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = httptest.NewRequest("DELETE", "/projects/test-uid", nil)
				ctx.Params = gin.Params{{Key: "projectId", Value: "test-uid"}}
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

			mockProjectRepo := mocks.NewMockProjectRepoClient(ctrl)
			mockServiceRepo := mocks.NewMockServiceRepoClient(ctrl)
			mockWebhookRepo := mocks.NewMockWebhookRepoClient(ctrl)
			mockWebhookService := mocks.NewMockWebhookEventSender(ctrl)

			service := NewProject(
				logger,
				mockProjectRepo,
				mockServiceRepo,
				mockWebhookRepo,
				restErr,
				config,
				mockWebhookService,
			)

			if tt.Prepare != nil {
				tt.Prepare(
					mockProjectRepo,
					mockServiceRepo,
					mockWebhookRepo,
					mockWebhookService,
				)
			}

			resp, err := service.DeleteProject(tt.Ctx())

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
