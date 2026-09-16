package object

import domain "asana/internal/domain/object"

type Repository struct {
	outputDir string
}

func New(outputDir string) *Repository {
	return &Repository{outputDir: outputDir}
}

type UserObject struct {
	GID          string              `json:"gid"`
	ResourceType domain.ResourceType `json:"resource_type"`
	Name         string              `json:"name"`
}

type ProjectObject struct {
	GID             string              `json:"gid"`
	ResourceType    domain.ResourceType `json:"resource_type"`
	ResourceSubType string              `json:"resource_subtype,omitempty"`
	Name            string              `json:"name"`
}
