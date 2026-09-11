package object

import (
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"

	domain "asana/internal/domain/object"
)

func TestExtractOnceFetchesUsersAndProjectsThenSaves(t *testing.T) {
	users := []domain.Object{{GID: "u1", ResourceType: domain.User, Name: "Ada"}}
	projects := []domain.Object{{GID: "p1", ResourceType: domain.Project, Name: "Launch"}}
	repo := &fakeRepository{}
	httpClient := &fakeHTTPClient{users: users, projects: projects}

	object := New(Config{PageLimit: 42}, httpClient, repo)

	if err := object.ExtractOnce(context.Background()); err != nil {
		t.Fatalf("ExtractOnce returned error: %v", err)
	}

	if !reflect.DeepEqual(httpClient.limits(), []int{42, 42}) {
		t.Fatalf("limits = %v, want [42 42]", httpClient.limits())
	}
	wantSaved := append(append([]domain.Object{}, users...), projects...)
	if !reflect.DeepEqual(repo.savedObjects(), wantSaved) {
		t.Fatalf("saved = %+v, want %+v", repo.savedObjects(), wantSaved)
	}
}

func TestExtractOnceDoesNotSaveWhenFetchFails(t *testing.T) {
	repo := &fakeRepository{}
	httpClient := &fakeHTTPClient{usersErr: errors.New("users unavailable")}
	object := New(Config{PageLimit: 100}, httpClient, repo)

	if err := object.ExtractOnce(context.Background()); err == nil {
		t.Fatal("ExtractOnce returned nil error, want fetch error")
	}
	if repo.saveCalls() != 0 {
		t.Fatalf("save calls = %d, want 0", repo.saveCalls())
	}
}

func TestExtractOnceFetchesUsersAndProjectsConcurrently(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	httpClient := &blockingHTTPClient{started: started, release: release}
	repo := &fakeRepository{}
	object := New(Config{PageLimit: 100}, httpClient, repo)

	errCh := make(chan error, 1)
	go func() {
		errCh <- object.ExtractOnce(context.Background())
	}()

	first := <-started
	second := <-started
	if first == second {
		t.Fatalf("expected both fetch methods to start, got %q and %q", first, second)
	}

	close(release)

	if err := <-errCh; err != nil {
		t.Fatalf("ExtractOnce returned error: %v", err)
	}
}

func TestRunContextStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repo := &fakeRepository{onSave: cancel}
	httpClient := &fakeHTTPClient{}
	object := New(
		Config{Interval: time.Hour, PageLimit: 100},
		httpClient,
		repo,
		WithLogger(log.New(io.Discard, "", 0)),
	)

	if err := object.RunContext(ctx); err != nil {
		t.Fatalf("RunContext returned error: %v", err)
	}
	if repo.saveCalls() != 1 {
		t.Fatalf("save calls = %d, want 1", repo.saveCalls())
	}
}

type fakeHTTPClient struct {
	mu          sync.Mutex
	users       []domain.Object
	projects    []domain.Object
	usersErr    error
	projectsErr error
	seenLimits  []int
}

func (client *fakeHTTPClient) FetchUsers(ctx context.Context, limit int) ([]domain.Object, error) {
	client.mu.Lock()
	client.seenLimits = append(client.seenLimits, limit)
	client.mu.Unlock()

	return client.users, client.usersErr
}

func (client *fakeHTTPClient) FetchProjects(ctx context.Context, limit int) ([]domain.Object, error) {
	client.mu.Lock()
	client.seenLimits = append(client.seenLimits, limit)
	client.mu.Unlock()

	return client.projects, client.projectsErr
}

func (client *fakeHTTPClient) limits() []int {
	client.mu.Lock()
	defer client.mu.Unlock()

	return append([]int{}, client.seenLimits...)
}

type blockingHTTPClient struct {
	started chan<- string
	release <-chan struct{}
}

func (client *blockingHTTPClient) FetchUsers(ctx context.Context, limit int) ([]domain.Object, error) {
	client.started <- "users"
	<-client.release
	return []domain.Object{{GID: "u1", ResourceType: domain.User}}, nil
}

func (client *blockingHTTPClient) FetchProjects(ctx context.Context, limit int) ([]domain.Object, error) {
	client.started <- "projects"
	<-client.release
	return []domain.Object{{GID: "p1", ResourceType: domain.Project}}, nil
}

type fakeRepository struct {
	mu     sync.Mutex
	saved  []domain.Object
	calls  int
	onSave func()
}

func (repo *fakeRepository) SaveObjects(objects []domain.Object) (string, error) {
	repo.mu.Lock()
	repo.calls++
	repo.saved = append([]domain.Object{}, objects...)
	onSave := repo.onSave
	repo.mu.Unlock()

	if onSave != nil {
		onSave()
	}

	return "output/asana_objects.json", nil
}

func (repo *fakeRepository) savedObjects() []domain.Object {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	return append([]domain.Object{}, repo.saved...)
}

func (repo *fakeRepository) saveCalls() int {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	return repo.calls
}
