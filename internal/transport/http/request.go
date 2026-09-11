package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	domain "asana/internal/domain/object"
	"asana/pkg/limiter"
	"asana/pkg/retry"
)

const (
	asanaEndpointPrefix    = "https://app.asana.com/api/1.0"
	getAllProjectsEndpoint = "projects"
	getAllUsersEndpoint    = "users"
)

type Config struct {
	AccessToken      string `yaml:"access_token"`
	RequestPerMinute int    `yaml:"request_per_minute"`
	RetryAfter       int    `yaml:"retry_after"`
	RetryAttempts    int    `yaml:"retry_attempts"`
}

type Client struct {
	baseURL      string
	httpClient   *nethttp.Client
	config       Config
	limiter      *limiter.Limiter
	sleep        retry.SleepFunc
	maxBodyInErr int64
}

type Option func(*Client)

func WithBaseURL(baseURL string) Option {
	return func(client *Client) {
		client.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithHTTPClient(httpClient *nethttp.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithSleeper(sleep retry.SleepFunc) Option {
	return func(client *Client) {
		if sleep != nil {
			client.sleep = sleep
		}
	}
}

func NewClient(config Config, options ...Option) *Client {
	client := &Client{
		baseURL:      asanaEndpointPrefix,
		httpClient:   &nethttp.Client{Timeout: 30 * time.Second},
		config:       config,
		limiter:      limiter.New(config.RequestPerMinute),
		sleep:        retry.Sleep,
		maxBodyInErr: 4096,
	}

	for _, option := range options {
		option(client)
	}

	return client
}

func (client *Client) FetchUsers(ctx context.Context, _ int) ([]domain.Object, error) {
	return client.fetchObjects(ctx, getAllUsersEndpoint, domain.User)
}

func (client *Client) FetchProjects(ctx context.Context, _ int) ([]domain.Object, error) {
	return client.fetchObjects(ctx, getAllProjectsEndpoint, domain.Project)
}

func (client *Client) fetchObjects(ctx context.Context, endpoint string, resourceType domain.ResourceType) ([]domain.Object, error) {
	page, err := client.fetchPage(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	objects := make([]domain.Object, 0)
	for _, item := range page.Data {
		objects = append(objects, domain.Object{
			GID:             item.GID,
			ResourceType:    normalizeResourceType(item.ResourceType, resourceType),
			ResourceSubType: item.ResourceSubType,
			Name:            item.Name,
		})
	}

	return objects, nil
}

func normalizeResourceType(value string, fallback domain.ResourceType) domain.ResourceType {
	if value == "" {
		return fallback
	}
	return domain.ResourceType(value)
}

func (client *Client) fetchPage(ctx context.Context, endpoint string) (asanaPage, error) {
	var page asanaPage
	err := retry.Do(ctx, client.config.RetryAttempts, time.Duration(client.config.RetryAfter)*time.Second, client.sleep, func(attempt int) (bool, time.Duration, error) {
		if err := client.limiter.Wait(ctx); err != nil {
			return false, 0, err
		}

		requestURL, err := client.buildURL(endpoint)
		if err != nil {
			return false, 0, err
		}

		request, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, requestURL, nil)
		if err != nil {
			return false, 0, err
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", "Bearer "+client.config.AccessToken)

		response, err := client.httpClient.Do(request)
		if err != nil {
			return true, 0, fmt.Errorf("request %s: %w", endpoint, err)
		}
		defer response.Body.Close()

		if response.StatusCode == nethttp.StatusOK {
			if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
				return false, 0, fmt.Errorf("decode %s response: %w", endpoint, err)
			}
			return false, 0, nil
		}

		body, _ := io.ReadAll(io.LimitReader(response.Body, client.maxBodyInErr))
		err = fmt.Errorf("asana %s returned status %d: %s", endpoint, response.StatusCode, strings.TrimSpace(string(body)))
		if response.StatusCode == nethttp.StatusTooManyRequests {
			return true, retryAfter(response.Header.Get("Retry-After")), err
		}
		if response.StatusCode >= 500 {
			return true, 0, err
		}

		return false, 0, err
	})
	if err != nil {
		return asanaPage{}, err
	}

	return page, nil
}

func (client *Client) buildURL(endpoint string) (string, error) {
	parsed, err := url.Parse(client.baseURL + "/" + strings.TrimLeft(endpoint, "/"))
	if err != nil {
		return "", err
	}

	return parsed.String(), nil
}

func retryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

type asanaPage struct {
	Data []asanaObject `json:"data"`
}

type asanaObject struct {
	GID             string `json:"gid"`
	ResourceType    string `json:"resource_type"`
	ResourceSubType string `json:"resource_subtype,omitempty"`
	Name            string `json:"name"`
}
