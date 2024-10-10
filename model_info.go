package huggingface

import (
	"fmt"
	"encoding/json"
	"net/http"
)

// RepoSibling represents a sibling file/directory in a repository.
type RepoSibling struct {
	Rfilename string `json:"rfilename"`
}

// ModelInfo represents information about a model repository.
type ModelInfo struct {
	ID       string         `json:"id"`
	Siblings []RepoSibling `json:"siblings"`
}

// ModelInfo retrieves information about a specific model repository.
func (c *HubClient) ModelInfo(repoId string) (*ModelInfo, error) {
	path := fmt.Sprintf("%s/api/models/%s", c.BaseURL, repoId)

	resp, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("model info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, CreateApiError(resp)
	}

	var modelInfo ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
		return nil, fmt.Errorf("error decoding model info: %w", err)
	}

	return &modelInfo, nil
}