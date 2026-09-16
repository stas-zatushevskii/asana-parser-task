package object

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	domain "asana/internal/domain/object"
)

func (repo *Repository) SaveObjects(objects []domain.Object) (string, error) {
	if repo.outputDir == "" {
		return "", fmt.Errorf("output directory is empty")
	}

	for _, obj := range objects {
		var payload any
		var path string

		switch obj.ResourceType {
		case domain.User:
			payload = UserObject{
				GID:          obj.GID,
				ResourceType: obj.ResourceType,
				Name:         obj.Name,
			}
			path = filepath.Join(repo.outputDir, "users", fmt.Sprintf("user_%s.json", obj.GID))
		case domain.Project:
			payload = ProjectObject{
				GID:             obj.GID,
				ResourceType:    obj.ResourceType,
				ResourceSubType: obj.ResourceSubType,
				Name:            obj.Name,
			}
			path = filepath.Join(repo.outputDir, "projects", fmt.Sprintf("project_%s.json", obj.GID))
		default:
			return "", fmt.Errorf("unsupported resource type %q for gid %q", obj.ResourceType, obj.GID)
		}

		if err := writeJSONFile(path, payload); err != nil {
			return "", err
		}
	}

	return repo.outputDir, nil
}

func writeJSONFile(path string, payload any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".asana-object-*.json")
	if err != nil {
		return fmt.Errorf("create temp output file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("encode output json: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp output file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace output file: %w", err)
	}

	return nil
}
