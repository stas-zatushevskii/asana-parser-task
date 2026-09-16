package object

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	domain "asana/internal/domain/object"
)

func TestSaveObjectsWritesSeparateJSONFiles(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "nested")
	repo := New(outputDir)

	writtenPath, err := repo.SaveObjects([]domain.Object{
		{
			GID:          "u1",
			ResourceType: domain.User,
			Name:         "Ada",
		},
		{
			GID:             "p1",
			ResourceType:    domain.Project,
			ResourceSubType: "default_project",
			Name:            "Launch",
		},
	})
	if err != nil {
		t.Fatalf("SaveObjects returned error: %v", err)
	}
	if writtenPath != outputDir {
		t.Fatalf("writtenPath = %q, want %q", writtenPath, outputDir)
	}

	userPath := filepath.Join(outputDir, "users", "user_u1.json")
	raw, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("ReadFile user returned error: %v", err)
	}
	var user UserObject
	if err := json.Unmarshal(raw, &user); err != nil {
		t.Fatalf("Unmarshal user returned error: %v", err)
	}
	if user.GID != "u1" || user.Name != "Ada" || user.ResourceType != domain.User {
		t.Fatalf("unexpected user payload: %+v", user)
	}

	projectPath := filepath.Join(outputDir, "projects", "project_p1.json")
	raw, err = os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("ReadFile project returned error: %v", err)
	}
	var project ProjectObject
	if err := json.Unmarshal(raw, &project); err != nil {
		t.Fatalf("Unmarshal project returned error: %v", err)
	}
	if project.GID != "p1" || project.ResourceSubType != "default_project" || project.ResourceType != domain.Project {
		t.Fatalf("unexpected project payload: %+v", project)
	}
}

func TestSaveObjectsRejectsUnknownResourceType(t *testing.T) {
	repo := New(t.TempDir())

	_, err := repo.SaveObjects([]domain.Object{{GID: "x1", ResourceType: "task"}})
	if err == nil {
		t.Fatal("SaveObjects returned nil error, want unsupported resource type error")
	}
}
