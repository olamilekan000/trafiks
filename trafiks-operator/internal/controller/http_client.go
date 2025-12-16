package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type HTTPClient interface {
	SetBaseURL(baseURL string) HTTPClient
	CheckHealth(ctx context.Context) (int, error)
	CheckAuthentication(ctx context.Context, apiKey string) (int, error)
}

type TrafiksHTTPClient struct {
	client  *resty.Client
	baseURL string
}

func NewTrafiksHTTPClient(timeout time.Duration) HTTPClient {
	client := resty.New().
		SetTimeout(timeout).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(3 * time.Second)

	return &TrafiksHTTPClient{
		client: client,
	}
}

func (c *TrafiksHTTPClient) SetBaseURL(baseURL string) HTTPClient {
	c.baseURL = baseURL
	return c
}

func (c *TrafiksHTTPClient) CheckHealth(ctx context.Context) (int, error) {
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

func (c *TrafiksHTTPClient) CheckAuthentication(ctx context.Context, apiKey string) (int, error) {
	if c.baseURL == "" {
		return 0, fmt.Errorf("baseURL not set")
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("X-API-Key", apiKey).
		Get(c.baseURL + "/users/data")

	if err != nil {
		return 0, err
	}

	return resp.StatusCode(), nil
}
