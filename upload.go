package huggingface

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

// UploadFile uploads a file to the specified repository
func (c *HubClient) UploadFile(repoName, repoType, filePath string) error {
	file, fileSize, fileName, err := openFile(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 1. Pre-upload (determine if regular or LFS)
	sample, _ := getFileSample(filePath)
	preuploadData, err := c.preUpload(repoName, repoType, fileName, fileSize, sample)
	if err != nil {
		return err
	}

	// 2. Check if file should be ignored
	if preuploadData.Files[0].ShouldIgnore {
		fmt.Printf("Skipping file: %s (marked as shouldIgnore)\n", fileName)
		return nil
	}

	// 3. Upload or Commit the file based on the mode
	switch preuploadData.Files[0].UploadMode {
	case "lfs":
		if err := c.uploadLFS(repoName, repoType, fileName, file); err != nil {
			return err
		}
		return c.commitFile(repoName, repoType, preuploadData.CommitOid, "upload file", fileName, file, true)
	case "regular":
		return c.commitFile(repoName, repoType, preuploadData.CommitOid, "upload file", fileName, file, false)
	default:
		return fmt.Errorf("unknown upload mode: %s", preuploadData.Files[0].UploadMode)
	}
}

// openFile opens a file and returns the file object, its size, and name
func openFile(filePath string) (*os.File, int64, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("error opening file: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, "", fmt.Errorf("error getting file info: %w", err)
	}

	return file, fileInfo.Size(), filepath.Base(filePath), nil
}

// PreuploadResponse defines the response from the pre-upload request
type PreuploadResponse struct {
	Files []struct {
		Path        string `json:"path"`
		UploadMode  string `json:"uploadMode"`
		ShouldIgnore bool   `json:"shouldIgnore"`
	} `json:"files"`
	CommitOid string `json:"commitOid"`
}

// preUpload sends the pre-upload request and parses the response
func (c *HubClient) preUpload(repoName, repoType, fileName string, fileSize int64, sample string) (*PreuploadResponse, error) {
	preuploadBody := map[string]interface{}{
		"files": []map[string]interface{}{
			{
				"path":   fileName,
				"size":   fileSize,
				"sample": sample,
			},
		},
	}
	preuploadURL := fmt.Sprintf("/api/%ss/%s/preupload/main", repoType, repoName)

	preuploadResp, err := c.doRequest("POST", preuploadURL, preuploadBody, nil)
	if err != nil {
		return nil, fmt.Errorf("pre-upload request failed: %w", err)
	}
	defer preuploadResp.Body.Close()

	if preuploadResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pre-upload failed: status code %d", preuploadResp.StatusCode)
	}

	var preuploadData PreuploadResponse
	if err := json.NewDecoder(preuploadResp.Body).Decode(&preuploadData); err != nil {
		return nil, fmt.Errorf("error decoding pre-upload response: %w", err)
	}

	return &preuploadData, nil
}


// ploadLFS handles the upload of a file to LFS
func (c *HubClient) uploadLFS(repoName, repoType, fileName string, fileData io.Reader) error {
    // 1. Calculate SHA256 Hash
    fileContent, err := ioutil.ReadAll(fileData)
    if err != nil {
        return fmt.Errorf("error reading file data: %w", err)
    }
    fileHash := sha256.Sum256(fileContent)
    hashString := fmt.Sprintf("%x", fileHash)

    // 2. Make LFS Batch Request
    lfsBatchBody := map[string]interface{}{
        "operation": "upload",
        "transfers": []string{"basic"}, // We can add "multipart" for larger files later
		"hash_algo": "sha_256",
		"ref": map[string]interface{}{"name": "main"},
        "objects": []map[string]interface{}{
            {
                "oid":  hashString,
                "size": len(fileContent),
            },
        },
    }
	lfsBatchURL := fmt.Sprintf("/%s.git/info/lfs/objects/batch",repoName)

    lfsBatchResp, err := c.doRequest("POST", lfsBatchURL, lfsBatchBody, lfsHeaders())
    if err != nil {
        return fmt.Errorf("LFS batch request failed: %w", err)
    }
    defer lfsBatchResp.Body.Close()

    // 3. Parse Batch Response and Upload
	return c.handleLFSResponse(lfsBatchResp, fileName, hashString, fileContent)
}

// lfsHeaders returns the headers required for LFS requests
func lfsHeaders() map[string]string {
	return map[string]string{
		"Accept":       "application/vnd.git-lfs+json",
		"Content-Type": "application/vnd.git-lfs+json",
	}
}

// LFSBatchResponse defines the response structure from an LFS batch request
type LFSBatchResponse struct {
	Objects []LFSObject `json:"objects"`
}

// LFSObject defines an individual object in an LFS batch request
type LFSObject struct {
	Oid     string `json:"oid"`
	Error   *struct { // Include Error struct
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Actions *struct { // Pointer to handle missing actions
		Upload *struct {
			Href string `json:"href"`
		} `json:"upload"`
		Verify *struct {
			Href   string            `json:"href"`
			Header map[string]string `json:"header"`
		} `json:"verify"`
	} `json:"actions"`
}

// handleLFSResponse processes the LFS batch response and uploads the file
func (c *HubClient) handleLFSResponse(resp *http.Response, fileName, hashString string, fileContent []byte) error {
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("LFS batch API request failed: %s (Status Code: %d)", string(responseBody), resp.StatusCode)
	}

	var batchResponse LFSBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResponse); err != nil {
		return fmt.Errorf("error decoding LFS batch response: %w", err)
	}

	if len(batchResponse.Objects) == 0 {
		return fmt.Errorf("LFS batch response contains no objects")
	}

	lfsObject := batchResponse.Objects[0]
	if lfsObject.Error != nil {
		return fmt.Errorf("LFS batch error: %s (code %d)", lfsObject.Error.Message, lfsObject.Error.Code)
	}

	if lfsObject.Actions == nil || lfsObject.Actions.Upload == nil {
		fmt.Printf("File %s already exists in LFS (oid: %s)\n", fileName, hashString)
		return nil
	}

	return c.uploadAndVerifyLFS(lfsObject, fileContent, hashString)
}

// uploadAndVerifyLFS uploads the file to the provided LFS URL and verifies it
func (c *HubClient) uploadAndVerifyLFS(lfsObject LFSObject, fileContent []byte, hashString string) error {
	uploadURL := lfsObject.Actions.Upload.Href

	if err := c.uploadFileToLFS(uploadURL, fileContent); err != nil {
		return err
	}

	if lfsObject.Actions.Verify != nil {
		verifyURL := lfsObject.Actions.Verify.Href
		verifyBody := map[string]interface{}{
			"oid": hashString, 
			"size": len(fileContent),
		}

		return c.verifyLFSUpload(verifyURL, verifyBody)
	}

	return nil
}

// uploadFileToLFS uploads the file content to the provided URL
func (c *HubClient) uploadFileToLFS(uploadURL string, fileContent []byte) error {
	req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(fileContent))
	if err != nil {
		return fmt.Errorf("error creating upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("LFS upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("LFS upload failed: %s", string(responseBody))
	}

	return nil
}

// verifyLFSUpload verifies the LFS upload by sending a verification request
func (c *HubClient) verifyLFSUpload(verifyURL string, verifyBody map[string]interface{}) error {
	verifyResp, err := c.doRequest("POST", verifyURL, verifyBody, nil)
	if err != nil {
		return fmt.Errorf("LFS verify request failed: %w", err)
	}
	defer verifyResp.Body.Close()

	if verifyResp.StatusCode != http.StatusOK {
		responseBody, _ := ioutil.ReadAll(verifyResp.Body)
		return fmt.Errorf("LFS verify failed: status code %d, body: %s", verifyResp.StatusCode, responseBody)
	}

	return nil
}


// commitFile commits a single file directly to the repository.
func (c *HubClient) commitFile(repoName, repoType, commitOid, commitMessage, fileName string, fileData io.Reader, isLFS bool) error {
	var operationsJSON []byte
    var fileSize int64

	if isLFS {
		if _, err := fileData.(*os.File).Seek(0, io.SeekStart); err != nil { // Check if fileData is *os.File
            return fmt.Errorf("error seeking file for LFS hash: %w", err)
        }
		fileContent, err := ioutil.ReadAll(fileData)
        if err != nil {
            return fmt.Errorf("error reading file data for LFS commit: %w", err)
        }

		fileHash := sha256.Sum256(fileContent)
        hashString := fmt.Sprintf("%x", fileHash)
        fileSize = int64(len(fileContent))

		operationsJSON, _ = json.Marshal(map[string]interface{}{
            "key": "lfsFile",
            "value": map[string]interface{}{
                "path": fileName,
                "algo": "sha256",
                "size": fileSize,
                "oid":  hashString,
            },
        })
	} else {
		// Encode the file data as base64
		fileContent, err := ioutil.ReadAll(fileData)
		if err != nil {
			return fmt.Errorf("error reading file data: %w", err)
		}
		base64Content := base64.StdEncoding.EncodeToString(fileContent)

		operationsJSON, _ = json.Marshal(map[string]interface{}{
			"key": "file",
			"value": map[string]interface{}{ // "value" is a single object now
				"path":     fileName,
				"encoding": "base64",
				"content":  base64Content,
			},
		})
	}

	// Commit the file
	headerJSON, _ := json.Marshal(map[string]interface{}{
		"key": "header",
		"value": map[string]interface{}{
			"summary": commitMessage,
			"parentCommit": commitOid,
		},
	})

	// Create NDJSON Request Body
	requestBody := []byte(string(headerJSON) + "\n\n" + string(operationsJSON))
	commitURL := fmt.Sprintf("%s/api/%ss/%s/commit/main", c.BaseURL, repoType, repoName)

	commitReq, err := http.NewRequest("POST", commitURL, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("error creating commit request: %w", err)
	}
	commitReq.Header.Set("Authorization", c.Auth.Header())
	commitReq.Header.Set("Content-Type", "application/x-ndjson") 

	commitResp, err := c.HTTPClient.Do(commitReq)
	if err != nil {
		return fmt.Errorf("commit request failed: %w", err)
	}
	defer commitResp.Body.Close()

	if commitResp.StatusCode != http.StatusOK {
		responseBody, _ := ioutil.ReadAll(commitResp.Body)
		fmt.Println("Commit Response Body:", string(responseBody))
		return fmt.Errorf("commit failed: status code %d", commitResp.StatusCode)
	}

	fmt.Printf("File %s committed successfully!\n", fileName)
	return nil
}


// Helper function to get base64 encoded sample of the file
func getFileSample(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	sampleBytes := make([]byte, 512) // Read up to 512 bytes
	n, err := file.Read(sampleBytes)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("error reading file sample: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sampleBytes[:n]), nil
}