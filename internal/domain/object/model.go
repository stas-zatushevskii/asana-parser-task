package object

type Object struct {
	GID             string       `json:"gid"`
	ResourceType    ResourceType `json:"resource_type"`
	ResourceSubType string       `json:"resource_subtype,omitempty"`
	Name            string       `json:"name"`
}

type ResourceType string

const (
	Project ResourceType = "project"
	User    ResourceType = "user"
)
