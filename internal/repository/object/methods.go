package object

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	domain "asana/internal/domain/object"
)

func (repo *Repository) SaveObjects(objects []domain.Object) (string, error) {
	if repo.outputPath == "" {
		return "", fmt.Errorf("output path is empty")
	}

	payload := FileStructure{
		Users:    make([]UserObject, 0),
		Projects: make([]ProjectObject, 0),
	}

	for _, obj := range objects {
		switch obj.ResourceType {
		case domain.User:
			payload.Users = append(payload.Users, UserObject{
				GID:          obj.GID,
				ResourceType: obj.ResourceType,
				Name:         obj.Name,
			})
		case domain.Project:
			payload.Projects = append(payload.Projects, ProjectObject{
				GID:             obj.GID,
				ResourceType:    obj.ResourceType,
				ResourceSubType: obj.ResourceSubType,
				Name:            obj.Name,
			})
		default:
			return "", fmt.Errorf("unsupported resource type %q for gid %q", obj.ResourceType, obj.GID)
		}
	}

	if err := os.MkdirAll(filepath.Dir(repo.outputPath), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(repo.outputPath), ".asana-objects-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp output file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("encode output json: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp output file: %w", err)
	}

	if err := os.Rename(tmpPath, repo.outputPath); err != nil {
		return "", fmt.Errorf("replace output file: %w", err)
	}

	return repo.outputPath, nil
}
