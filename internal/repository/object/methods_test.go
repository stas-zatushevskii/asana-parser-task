package object

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	domain "asana/internal/domain/object"
)

func TestSaveObjectsWritesGroupedJSON(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "nested", "asana_objects.json")
	repo := New(outputPath)

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
	if writtenPath != outputPath {
		t.Fatalf("writtenPath = %q, want %q", writtenPath, outputPath)
	}

	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	var payload FileStructure
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if len(payload.Users) != 1 {
		t.Fatalf("users len = %d, want 1", len(payload.Users))
	}
	if payload.Users[0].GID != "u1" || payload.Users[0].Name != "Ada" || payload.Users[0].ResourceType != domain.User {
		t.Fatalf("unexpected user payload: %+v", payload.Users[0])
	}
	if len(payload.Projects) != 1 {
		t.Fatalf("projects len = %d, want 1", len(payload.Projects))
	}
	if payload.Projects[0].GID != "p1" || payload.Projects[0].ResourceSubType != "default_project" || payload.Projects[0].ResourceType != domain.Project {
		t.Fatalf("unexpected project payload: %+v", payload.Projects[0])
	}
}

func TestSaveObjectsRejectsUnknownResourceType(t *testing.T) {
	repo := New(filepath.Join(t.TempDir(), "asana_objects.json"))

	_, err := repo.SaveObjects([]domain.Object{{GID: "x1", ResourceType: "task"}})
	if err == nil {
		t.Fatal("SaveObjects returned nil error, want unsupported resource type error")
	}
}
