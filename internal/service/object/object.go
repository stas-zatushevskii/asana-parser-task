package object

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	domain "asana/internal/domain/object"
)

type HTTPClient interface {
	FetchUsers(ctx context.Context, limit int) ([]domain.Object, error)
	FetchProjects(ctx context.Context, limit int) ([]domain.Object, error)
}

type Repository interface {
	SaveObjects(objects []domain.Object) (string, error)
}

type Logger interface {
	Printf(format string, args ...any)
}

type Config struct {
	Interval  time.Duration
	PageLimit int
}

type Object struct {
	config     Config
	httpClient HTTPClient
	repository Repository
	logger     Logger
}

type Option func(*Object)

func WithLogger(logger Logger) Option {
	return func(object *Object) {
		if logger != nil {
			object.logger = logger
		}
	}
}

func New(config Config, httpClient HTTPClient, repository Repository, options ...Option) *Object {
	object := &Object{
		config:     config,
		httpClient: httpClient,
		repository: repository,
		logger:     log.Default(),
	}

	for _, option := range options {
		option(object)
	}

	return object
}

func (object *Object) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return object.RunContext(ctx)
}

func (object *Object) RunContext(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}
	if object.config.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}

	if err := object.ExtractOnce(ctx); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		ticker := time.NewTicker(object.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := object.ExtractOnce(ctx); err != nil {
					object.logger.Printf("extract failed: %v", err)
				}
			}
		}
	}()

	<-ctx.Done()
	<-done

	return nil
}

func (object *Object) ExtractOnce(ctx context.Context) error {
	if object.httpClient == nil {
		return fmt.Errorf("http client is nil")
	}
	if object.repository == nil {
		return fmt.Errorf("repository is nil")
	}

	type result struct {
		objects []domain.Object
		err     error
	}

	usersCh := make(chan result, 1)
	projectsCh := make(chan result, 1)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		users, err := object.httpClient.FetchUsers(ctx, object.config.PageLimit)
		usersCh <- result{objects: users, err: err}
	}()

	go func() {
		defer wg.Done()
		projects, err := object.httpClient.FetchProjects(ctx, object.config.PageLimit)
		projectsCh <- result{objects: projects, err: err}
	}()

	wg.Wait()
	close(usersCh)
	close(projectsCh)

	usersResult := <-usersCh
	if usersResult.err != nil {
		return fmt.Errorf("fetch users: %w", usersResult.err)
	}

	projectsResult := <-projectsCh
	if projectsResult.err != nil {
		return fmt.Errorf("fetch projects: %w", projectsResult.err)
	}

	objects := make([]domain.Object, 0, len(usersResult.objects)+len(projectsResult.objects))
	objects = append(objects, usersResult.objects...)
	objects = append(objects, projectsResult.objects...)

	if _, err := object.repository.SaveObjects(objects); err != nil {
		return fmt.Errorf("save objects: %w", err)
	}

	return nil
}
