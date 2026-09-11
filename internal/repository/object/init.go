package object

import domain "asana/internal/domain/object"

type Repository struct {
	outputPath string
}

func New(outputPath string) *Repository {
	return &Repository{outputPath: outputPath}
}

type FileStructure struct {
	Users    []UserObject    `json:"users"`
	Projects []ProjectObject `json:"projects"`
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
