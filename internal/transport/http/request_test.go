package http

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	domain "asana/internal/domain/object"
)

type fetchTarget string

const (
	targetUsers    fetchTarget = "users"
	targetProjects fetchTarget = "projects"
)

type apiStep struct {
	status  int
	headers map[string]string
	body    string
}

type capturedRequest struct {
	path      string
	auth      string
	accept    string
	rawQuery  string
	requestID int
}

type clientCase struct {
	name          string
	target        fetchTarget
	limit         int
	config        Config
	steps         []apiStep
	want          []domain.Object
	wantErr       bool
	wantAttempts  int
	wantSleeps    []time.Duration
	wantPath      string
	forbidSleeper bool
}

func TestClientResponses(t *testing.T) {
	tests := []clientCase{
		{
			name:   "200 users success",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 3, RetryAfter: 30},
			steps: []apiStep{
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{"gid": "u1", "resource_type": "user", "name": "Ada"},
					},
				}),
			},
			want: []domain.Object{
				{GID: "u1", ResourceType: domain.User, Name: "Ada"},
			},
			wantAttempts: 1,
			wantPath:     "/users",
		},
		{
			name:   "200 projects success",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 3, RetryAfter: 30},
			steps: []apiStep{
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{
							"gid":              "p1",
							"resource_type":    "project",
							"resource_subtype": "default_project",
							"name":             "Launch",
						},
					},
				}),
			},
			want: []domain.Object{
				{GID: "p1", ResourceType: domain.Project, ResourceSubType: "default_project", Name: "Launch"},
			},
			wantAttempts: 1,
			wantPath:     "/projects",
		},
		{
			name:   "200 malformed json does not retry",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 3, RetryAfter: 30},
			steps: []apiStep{
				{status: nethttp.StatusOK, body: `{"data":[`},
			},
			wantErr:       true,
			wantAttempts:  1,
			wantPath:      "/users",
			forbidSleeper: true,
		},
		{
			name:   "400 returns error without retry",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 3, RetryAfter: 30},
			steps: []apiStep{
				errorStep(nethttp.StatusBadRequest, "bad request"),
			},
			wantErr:       true,
			wantAttempts:  1,
			wantPath:      "/users",
			forbidSleeper: true,
		},
		{
			name:   "401 returns error without retry",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 3, RetryAfter: 30},
			steps: []apiStep{
				errorStep(nethttp.StatusUnauthorized, "unauthorized"),
			},
			wantErr:       true,
			wantAttempts:  1,
			wantPath:      "/users",
			forbidSleeper: true,
		},
		{
			name:   "403 returns error without retry",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 3, RetryAfter: 30},
			steps: []apiStep{
				errorStep(nethttp.StatusForbidden, "forbidden"),
			},
			wantErr:       true,
			wantAttempts:  1,
			wantPath:      "/projects",
			forbidSleeper: true,
		},
		{
			name:   "404 returns error without retry",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 3, RetryAfter: 30},
			steps: []apiStep{
				errorStep(nethttp.StatusNotFound, "not found"),
			},
			wantErr:       true,
			wantAttempts:  1,
			wantPath:      "/projects",
			forbidSleeper: true,
		},
		{
			name:   "429 retries with retry-after",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 30},
			steps: []apiStep{
				{
					status:  nethttp.StatusTooManyRequests,
					headers: map[string]string{"Retry-After": "1"},
					body:    `{"errors":[{"message":"rate limited"}]}`,
				},
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{"gid": "u1", "resource_type": "user", "name": "Ada"},
					},
				}),
			},
			want: []domain.Object{
				{GID: "u1", ResourceType: domain.User, Name: "Ada"},
			},
			wantAttempts: 2,
			wantSleeps:   []time.Duration{time.Second},
			wantPath:     "/users",
		},
		{
			name:   "429 without retry-after uses fallback",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 7},
			steps: []apiStep{
				errorStep(nethttp.StatusTooManyRequests, "rate limited"),
				jsonStep(nethttp.StatusOK, map[string]any{"data": []map[string]string{}}),
			},
			want:         []domain.Object{},
			wantAttempts: 2,
			wantSleeps:   []time.Duration{7 * time.Second},
			wantPath:     "/users",
		},
		{
			name:   "429 with invalid retry-after uses fallback",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 9},
			steps: []apiStep{
				{
					status:  nethttp.StatusTooManyRequests,
					headers: map[string]string{"Retry-After": "later"},
					body:    `{"errors":[{"message":"rate limited"}]}`,
				},
				jsonStep(nethttp.StatusOK, map[string]any{"data": []map[string]string{}}),
			},
			want:         []domain.Object{},
			wantAttempts: 2,
			wantSleeps:   []time.Duration{9 * time.Second},
			wantPath:     "/users",
		},
		{
			name:   "429 exhausted returns error",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 5},
			steps: []apiStep{
				errorStep(nethttp.StatusTooManyRequests, "rate limited"),
				errorStep(nethttp.StatusTooManyRequests, "still rate limited"),
			},
			wantErr:      true,
			wantAttempts: 2,
			wantSleeps:   []time.Duration{5 * time.Second},
			wantPath:     "/users",
		},
		{
			name:   "500 retries with fallback",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 7},
			steps: []apiStep{
				errorStep(nethttp.StatusInternalServerError, "temporary"),
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{"gid": "p1", "resource_type": "project", "name": "Launch"},
					},
				}),
			},
			want: []domain.Object{
				{GID: "p1", ResourceType: domain.Project, Name: "Launch"},
			},
			wantAttempts: 2,
			wantSleeps:   []time.Duration{7 * time.Second},
			wantPath:     "/projects",
		},
		{
			name:   "502 retries with fallback",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 6},
			steps: []apiStep{
				errorStep(nethttp.StatusBadGateway, "bad gateway"),
				jsonStep(nethttp.StatusOK, map[string]any{"data": []map[string]string{}}),
			},
			want:         []domain.Object{},
			wantAttempts: 2,
			wantSleeps:   []time.Duration{6 * time.Second},
			wantPath:     "/projects",
		},
		{
			name:   "503 retries with fallback",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 8},
			steps: []apiStep{
				errorStep(nethttp.StatusServiceUnavailable, "unavailable"),
				jsonStep(nethttp.StatusOK, map[string]any{"data": []map[string]string{}}),
			},
			want:         []domain.Object{},
			wantAttempts: 2,
			wantSleeps:   []time.Duration{8 * time.Second},
			wantPath:     "/projects",
		},
		{
			name:   "5xx exhausted returns error",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 2, RetryAfter: 4},
			steps: []apiStep{
				errorStep(nethttp.StatusInternalServerError, "temporary"),
				errorStep(nethttp.StatusServiceUnavailable, "still unavailable"),
			},
			wantErr:      true,
			wantAttempts: 2,
			wantSleeps:   []time.Duration{4 * time.Second},
			wantPath:     "/projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runClientCase(t, tt)

			if tt.wantErr && result.err == nil {
				t.Fatal("client returned nil error, want error")
			}
			if !tt.wantErr && result.err != nil {
				t.Fatalf("client returned error: %v", result.err)
			}
			if !reflect.DeepEqual(result.objects, tt.want) {
				t.Fatalf("objects = %+v, want %+v", result.objects, tt.want)
			}
			assertRequests(t, result.requests, tt)
			assertSleeps(t, result.sleeps, tt.wantSleeps)
		})
	}
}

func TestClientEndpointBehavior(t *testing.T) {
	tests := []clientCase{
		{
			name:   "users ignores next page",
			target: targetUsers,
			limit:  2,
			config: Config{AccessToken: "test-token", RetryAttempts: 1},
			steps: []apiStep{
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{"gid": "u1", "resource_type": "user", "name": "Ada"},
					},
					"next_page": map[string]string{"offset": "next-users"},
				}),
			},
			want: []domain.Object{
				{GID: "u1", ResourceType: domain.User, Name: "Ada"},
			},
			wantAttempts: 1,
			wantPath:     "/users",
		},
		{
			name:   "projects preserve resource subtype",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 1},
			steps: []apiStep{
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{
							"gid":              "p1",
							"resource_type":    "project",
							"resource_subtype": "default_project",
							"name":             "Launch",
						},
					},
				}),
			},
			want: []domain.Object{
				{GID: "p1", ResourceType: domain.Project, ResourceSubType: "default_project", Name: "Launch"},
			},
			wantAttempts: 1,
			wantPath:     "/projects",
		},
		{
			name:   "users missing resource type falls back to user",
			target: targetUsers,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 1},
			steps: []apiStep{
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{"gid": "u1", "name": "Ada"},
					},
				}),
			},
			want: []domain.Object{
				{GID: "u1", ResourceType: domain.User, Name: "Ada"},
			},
			wantAttempts: 1,
			wantPath:     "/users",
		},
		{
			name:   "projects missing resource type falls back to project",
			target: targetProjects,
			limit:  100,
			config: Config{AccessToken: "test-token", RetryAttempts: 1},
			steps: []apiStep{
				jsonStep(nethttp.StatusOK, map[string]any{
					"data": []map[string]string{
						{"gid": "p1", "name": "Launch"},
					},
				}),
			},
			want: []domain.Object{
				{GID: "p1", ResourceType: domain.Project, Name: "Launch"},
			},
			wantAttempts: 1,
			wantPath:     "/projects",
		},
		{
			name:   "limit argument is ignored",
			target: targetUsers,
			limit:  101,
			config: Config{AccessToken: "test-token", RetryAttempts: 1},
			steps: []apiStep{
				jsonStep(nethttp.StatusOK, map[string]any{"data": []map[string]string{}}),
			},
			want:         []domain.Object{},
			wantAttempts: 1,
			wantPath:     "/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runClientCase(t, tt)

			if tt.wantErr && result.err == nil {
				t.Fatal("client returned nil error, want error")
			}
			if !tt.wantErr && result.err != nil {
				t.Fatalf("client returned error: %v", result.err)
			}
			if !reflect.DeepEqual(result.objects, tt.want) {
				t.Fatalf("objects = %+v, want %+v", result.objects, tt.want)
			}
			assertRequests(t, result.requests, tt)
			assertSleeps(t, result.sleeps, tt.wantSleeps)
		})
	}
}

type clientCaseResult struct {
	objects  []domain.Object
	err      error
	requests []capturedRequest
	sleeps   []time.Duration
}

func runClientCase(t *testing.T, tt clientCase) clientCaseResult {
	t.Helper()

	if tt.config.AccessToken == "" {
		tt.config.AccessToken = "test-token"
	}

	requests := make([]capturedRequest, 0, len(tt.steps))
	server := httptest.NewServer(nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		requests = append(requests, capturedRequest{
			path:      request.URL.Path,
			auth:      request.Header.Get("Authorization"),
			accept:    request.Header.Get("Accept"),
			rawQuery:  request.URL.RawQuery,
			requestID: len(requests) + 1,
		})

		if len(requests) > len(tt.steps) {
			t.Fatalf("unexpected request %d to %s", len(requests), request.URL.String())
		}

		step := tt.steps[len(requests)-1]
		for key, value := range step.headers {
			writer.Header().Set(key, value)
		}
		if step.status == 0 {
			step.status = nethttp.StatusOK
		}
		writer.WriteHeader(step.status)
		_, _ = writer.Write([]byte(step.body))
	}))
	defer server.Close()

	sleeps := make([]time.Duration, 0)
	sleeper := func(ctx context.Context, delay time.Duration) error {
		if tt.forbidSleeper {
			t.Fatalf("unexpected sleep for non-retryable case: %s", delay)
		}
		sleeps = append(sleeps, delay)
		return nil
	}

	client := NewClient(
		tt.config,
		WithBaseURL(server.URL),
		WithSleeper(sleeper),
	)

	objects, err := fetch(t, client, tt.target, tt.limit)
	return clientCaseResult{
		objects:  objects,
		err:      err,
		requests: requests,
		sleeps:   sleeps,
	}
}

func fetch(t *testing.T, client *Client, target fetchTarget, limit int) ([]domain.Object, error) {
	t.Helper()

	switch target {
	case targetUsers:
		return client.FetchUsers(context.Background(), limit)
	case targetProjects:
		return client.FetchProjects(context.Background(), limit)
	default:
		t.Fatalf("unknown target %q", target)
		return nil, nil
	}
}

func assertRequests(t *testing.T, requests []capturedRequest, tt clientCase) {
	t.Helper()

	if len(requests) != tt.wantAttempts {
		t.Fatalf("request count = %d, want %d", len(requests), tt.wantAttempts)
	}

	for _, request := range requests {
		if request.auth != "Bearer "+tt.config.AccessToken {
			t.Fatalf("request %d Authorization = %q, want %q", request.requestID, request.auth, "Bearer "+tt.config.AccessToken)
		}
		if request.accept != "application/json" {
			t.Fatalf("request %d Accept = %q, want application/json", request.requestID, request.accept)
		}
		if request.path != tt.wantPath {
			t.Fatalf("request %d path = %q, want %q", request.requestID, request.path, tt.wantPath)
		}
		if request.rawQuery != "" {
			t.Fatalf("request %d raw query = %q, want empty query", request.requestID, request.rawQuery)
		}
	}
}

func assertSleeps(t *testing.T, sleeps []time.Duration, want []time.Duration) {
	t.Helper()

	if len(sleeps) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(sleeps, want) {
		t.Fatalf("sleeps = %v, want %v", sleeps, want)
	}
}

func jsonStep(status int, value any) apiStep {
	var builder strings.Builder
	if err := json.NewEncoder(&builder).Encode(value); err != nil {
		panic(err)
	}

	return apiStep{
		status:  status,
		headers: map[string]string{"Content-Type": "application/json"},
		body:    builder.String(),
	}
}

func errorStep(status int, message string) apiStep {
	return jsonStep(status, map[string]any{
		"errors": []map[string]string{
			{"message": message},
		},
	})
}
