package sync

// Contains logic to handle files

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	url := "https://bytebridge.es8.nl/api/v1/File"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
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

// CalculateFileHash computes the MD5 hash of a file, which is used to check if files are identical
func CalculateFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := fmt.Sprintf("%x", md5.Sum(data))
	return hash, nil
}

// FetchDeletedFiles retrieves the list of deleted files from the server
func FetchDeletedFiles() ([]File, error) {
	url := "https://bytebridge.es8.nl/api/v1/File/deleted"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	log.Println("FetchDeletedFiles API response status:", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Println("FetchDeletedFiles API response body:", string(body))

	var files []File
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return files, nil
}
