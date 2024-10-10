package huggingface

import (
	"fmt"
	"encoding/json"
	"net/http"
)

// FileInfo represents information about a file in a repository.
type FileInfo struct { 
	Type string `json:"type"`
	Oid string `json:"oid"`
	Size int `json:"size"`
	Path string `json:"path"`
}

// ListFiles lists all files in a repository.
func (c *HubClient) ListFiles(repoId string) (*[]FileInfo, error) {
	path := fmt.Sprintf("%s/api/models/%s/tree/main", c.BaseURL, repoId)

	resp, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("model info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, CreateApiError(resp)
	}

	// receive a list of objects like {"type":"file","oid":"1a42b08d14b65b4b977fd2f74b226c26f73eb9fc","size":1716,"path":".gitattributes"}
	var FileInfo []FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&FileInfo); err != nil {
		return nil, fmt.Errorf("error decoding model info: %w", err)
	}

	return &FileInfo, nil
}