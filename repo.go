package huggingface

import (
	"fmt"
	"net/http"
	"strings"
)

// CreateRepoOptions holds options for creating a repository
type CreateRepoOptions struct {
	ExistsOK bool `json:"exist_ok"`
	Private  bool `json:"private,omitempty"`
	License  string `json:"license,omitempty"`
	Sdk      string `json:"sdk,omitempty"` // e.g., for spaces
}

// CreateRepoResponse represents the response from creating a repo
type CreateRepoResponse struct {
	URL string `json:"url"`
}

// CreateRepo creates a new repository on Hugging Face Hub
func (c *HubClient) CreateRepo(repoId, repoType string, opts *CreateRepoOptions) (*CreateRepoResponse, error) {
	var respData CreateRepoResponse

	// Validate and split the repoId into namespace and repoName
	namespace, repoName, err := parseRepoID(repoId)
	if err != nil {
		return nil, err
	}

	// Build the payload using options
	payload := buildPayload(namespace, repoName, repoType, opts)

	// Send the request
	resp, err := c.doRequest("POST", "/api/repos/create", payload, nil)
	if err != nil {
		return nil, err
	}

	err = parseResponse(resp, &respData)
	if err != nil {
		// Ignore conflict if ExistsOK is true
		if opts != nil && opts.ExistsOK && resp != nil && resp.StatusCode == http.StatusConflict {
			return &respData, nil
		}
		return nil, err
	}

	return &respData, nil
}

// buildPayload constructs the payload for the API request
func buildPayload(namespace, repoName, repoType string, opts *CreateRepoOptions) map[string]interface{} {
	payload := map[string]interface{}{
		"name":         repoName,
		"organization": namespace,
		"type":         repoType,  // e.g., "model", "dataset", "space"
		"private":      false,     // Default to public
	}

	// If options are provided, apply them
	if opts != nil {
		if opts.Private {
			payload["private"] = opts.Private
		}
		if opts.License != "" {
			payload["license"] = opts.License
		}
		if opts.Sdk != "" {
			payload["sdk"] = opts.Sdk
		}
	}

	return payload
}

// parseRepoID validates and splits the repository ID into namespace and repoName
func parseRepoID(repoId string) (string, string, error) {
	parts := strings.Split(repoId, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository name format: %s (should be 'namespace/repoName')", repoId)
	}
	return parts[0], parts[1], nil
}