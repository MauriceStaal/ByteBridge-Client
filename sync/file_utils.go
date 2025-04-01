package sync

// Contains logic to handle files

import (
	"ByteBridge-Client/config"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// File represents the structure of a file from the API response
type File struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Hash      string `json:"hash"`
	Extension string `json:"extension"`
	CreatedOn string `json:"createdOn"`
	UpdatedOn string `json:"updatedOn"`
}

var uploadMutex sync.Mutex
var lastUploaded = make(map[string]time.Time)

// FetchFiles requests the list of files from the API and returns them
func FetchFiles() ([]File, error) {
	url := config.APIEndpoint("/File")
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var files []File
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return files, nil
}

// GetFileIDByName retrieves the file ID from the server by filename
func GetFileIDByName(filename string) (int, error) {
	files, err := FetchFiles()
	if err != nil {
		return 0, err
	}
	for _, file := range files {
		if file.Name == filename {
			return file.ID, nil
		}
	}
	return 0, fmt.Errorf("file ID not found for %s", filename)
}

// FileExists checks if a file exists in the sync folder
func FileExists(syncFolder, filename string) bool {
	filePath := filepath.Join(syncFolder, filename)
	_, err := os.Stat(filePath)
	return err == nil
}

// GetFileIDByHash retrieves the file ID from the server by file hash
func GetFileIDByHash(fileHash string) (int, error) {
	files, err := FetchFiles()
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		if file.Hash == fileHash { // Controleer op hash in plaats van naam
			return file.ID, nil
		}
	}

	return 0, fmt.Errorf("file with hash %s not found", fileHash)
}

// CalculateFileHash calculates the SHA-256 hash of a file
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// FetchDeletedFiles retrieves the list of deleted files from the server
func FetchDeletedFiles() ([]File, error) {
	url := config.APIEndpoint("/File/deleted")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	config.DebugLogger.Println("FetchDeletedFiles API response status:", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	config.DebugLogger.Println("FetchDeletedFiles API response body:", string(body))

	var files []File
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return files, nil
}
