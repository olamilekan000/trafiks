//go:generate mockgen -source=trafiks_api_client.go -destination=../test/mocks/trafiks_api_client.go -package=mocks

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type TrafiksAPIClient interface {
	SetAPIConfig(baseURL, apiKey string)

	CheckHealth(ctx context.Context) (int, error)
	CheckAuthentication(ctx context.Context) (int, error)

	ListProjects(ctx context.Context) ([]map[string]interface{}, error)
	GetProject(ctx context.Context, projectUID string) (map[string]interface{}, error)
	CreateProject(ctx context.Context, name, description string) (map[string]interface{}, error)

	GetService(ctx context.Context, projectUID, proxyURL string) (map[string]interface{}, error)
	CreateService(ctx context.Context, projectUID string, serviceData map[string]interface{}) (map[string]interface{}, error)
	UpdateService(ctx context.Context, projectUID, serviceUID string, serviceData map[string]interface{}) (map[string]interface{}, error)
	DeleteProject(ctx context.Context, projectUID string) error
}

type trafiksAPIClient struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

func NewTrafiksAPIClient(timeout time.Duration) TrafiksAPIClient {
	client := resty.New().
		SetTimeout(timeout).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(3 * time.Second)

	return &trafiksAPIClient{
		client: client,
	}
}

func (c *trafiksAPIClient) SetAPIConfig(baseURL, apiKey string) {
	c.baseURL = baseURL
	c.apiKey = apiKey
}

// CheckHealth checks if the Trafiks backend is reachable via /health endpoint
func (c *trafiksAPIClient) CheckHealth(ctx context.Context) (int, error) {
	if c.baseURL == "" {
		return 0, fmt.Errorf("baseURL not set")
	}

	resp, err := c.client.R().
		SetContext(ctx).
		Get(c.baseURL + "/health")

	if err != nil {
		return 0, err
	}

	return resp.StatusCode(), nil
}

// CheckAuthentication validates the API key by calling /users/data endpoint
func (c *trafiksAPIClient) CheckAuthentication(ctx context.Context) (int, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return 0, fmt.Errorf("baseURL and apiKey must be set")
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		Get(c.baseURL + "/users/data")

	if err != nil {
		return 0, err
	}

	return resp.StatusCode(), nil
}

func (c *trafiksAPIClient) ListProjects(ctx context.Context) ([]map[string]interface{}, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return nil, fmt.Errorf("baseURL and apiKey must be set")
	}

	var result struct {
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		SetResult(&result).
		Get(c.baseURL + "/projects")

	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to list projects: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned error: %s", result.Message)
	}

	if result.Data == nil {
		return []map[string]interface{}{}, nil
	}

	projectsRaw, ok := result.Data["projects"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	projects, ok := projectsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected projects format: expected array")
	}

	resultProjects := make([]map[string]interface{}, 0, len(projects))
	for _, p := range projects {
		if projectMap, ok := p.(map[string]interface{}); ok {
			resultProjects = append(resultProjects, projectMap)
		}
	}

	return resultProjects, nil
}

func (c *trafiksAPIClient) GetProject(ctx context.Context, projectUID string) (map[string]interface{}, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return nil, fmt.Errorf("baseURL and apiKey must be set")
	}

	var result struct {
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		SetResult(&result).
		Get(c.baseURL + "/projects/" + projectUID)

	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get project: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned error: %s", result.Message)
	}

	return result.Data, nil
}

func (c *trafiksAPIClient) DeleteProject(ctx context.Context, projectUID string) error {
	if c.baseURL == "" || c.apiKey == "" {
		return fmt.Errorf("baseURL and apiKey must be set")
	}

	var result struct {
		Message string `json:"message"`
		Success bool   `json:"success"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		SetResult(&result).
		Delete(c.baseURL + "/projects/" + projectUID)

	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	if resp.StatusCode() == 404 {
		return nil
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to delete project: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.Success {
		return fmt.Errorf("API returned error: %s", result.Message)
	}

	return nil
}

func (c *trafiksAPIClient) CreateProject(ctx context.Context, name, description string) (map[string]interface{}, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return nil, fmt.Errorf("baseURL and apiKey must be set")
	}

	var result struct {
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}

	payload := map[string]interface{}{
		"Name":        name,
		"Description": description,
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		SetBody(payload).
		SetResult(&result).
		Post(c.baseURL + "/projects")

	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create project: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned error: %s", result.Message)
	}

	return result.Data, nil
}

func (c *trafiksAPIClient) GetService(ctx context.Context, projectUID, proxyURL string) (map[string]interface{}, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return nil, fmt.Errorf("baseURL and apiKey must be set")
	}

	var result struct {
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		SetResult(&result).
		Get(c.baseURL + "/projects/" + projectUID + "/service")

	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("service not found with proxyURL: %s", proxyURL)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get service: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned error: %s", result.Message)
	}

	if result.Data != nil {
		svcProxyURL, ok := result.Data["proxy_url"].(string)
		if ok && svcProxyURL == proxyURL {
			return result.Data, nil
		}
		return nil, fmt.Errorf("service found but proxyURL mismatch: expected %s, got %s", proxyURL, svcProxyURL)
	}

	return nil, fmt.Errorf("service not found with proxyURL: %s", proxyURL)
}

func (c *trafiksAPIClient) CreateService(ctx context.Context, projectUID string, serviceData map[string]interface{}) (map[string]interface{}, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return nil, fmt.Errorf("baseURL and apiKey must be set")
	}

	var result struct {
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		SetBody(serviceData).
		SetResult(&result).
		Post(c.baseURL + "/projects/" + projectUID + "/services")

	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create service: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned error: %s", result.Message)
	}

	return result.Data, nil
}

func (c *trafiksAPIClient) UpdateService(ctx context.Context, projectUID, serviceUID string, serviceData map[string]interface{}) (map[string]interface{}, error) {
	if c.baseURL == "" || c.apiKey == "" {
		return nil, fmt.Errorf("baseURL and apiKey must be set")
	}

	var result struct {
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", c.apiKey).
		SetBody(serviceData).
		SetResult(&result).
		Put(c.baseURL + "/projects/" + projectUID + "/services/" + serviceUID)

	if err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to update service: status %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned error: %s", result.Message)
	}

	return result.Data, nil
}
