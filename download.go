package huggingface

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadFile downloads a file from the specified repository and path, and saves it to the given local path.
// Returns nil if successful, or an error if the download or file writing fails.
func (c *HubClient) DownloadFile(repoId, repoType, filePath, localFilePath string) error {
	// Determine the repository type path
	repoTypePath := ""
	if repoType != "model" {
		repoTypePath = fmt.Sprintf("%ss/", repoType)
	}

	// Construct the download URL
	downloadURL := fmt.Sprintf("%s/%s%s/resolve/main/%s", c.BaseURL, repoTypePath, repoId, filePath)

	// Make the request
	resp, err := c.doRequest("GET", downloadURL, nil, nil)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle 404 (File not found) specifically
	if resp.StatusCode == http.StatusNotFound {
		if resp.Header.Get("X-Error-Code") == "EntryNotFound" {
			return fmt.Errorf("file not found in repository: %s", filePath)
		}
	}

	// Handle any non-200 responses as errors
	if resp.StatusCode != http.StatusOK {
		return CreateApiError(resp)
	}

	// Ensure the directory exists before saving the file
	localDir := filepath.Dir(localFilePath)
	err = os.MkdirAll(localDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create or overwrite the local file
	outFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer outFile.Close()

	// Write the downloaded content to the local file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file content to local path: %w", err)
	}

	fmt.Printf("File downloaded and saved to %s\n", localFilePath)
	return nil
}
