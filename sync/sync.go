package sync

// Contains sync logic for checking and downloading files

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
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
	url := "http://localhost:5191/api/v1/File"
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

// SyncFiles checks if the files from the server exist on the client and downloads the missing ones
func SyncFiles(syncFolder string) {
	for {
		// Fetch the list of files from the server
		files, err := FetchFiles()
		if err != nil {
			fmt.Println("Error fetching files:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Check for each file if it exists on the client, and if not, download it
		for _, file := range files {
			if !FileExists(syncFolder, file.Name) {
				fmt.Println("File not found locally, downloading:", file.ID, file.Name)
				err := DownloadFile(syncFolder, file.ID, file.Name)
				if err != nil {
					fmt.Println("Error downloading file:", err)
				}
			} else {
				fmt.Println("File already exists locally:", file.Name)
			}
		}

		// Wait for 30 seconds before checking again
		time.Sleep(30 * time.Second)
	}
}

// DeleteFileOnServer deletes a file from the server
func DeleteFileOnServer(fileID int) error {
	url := fmt.Sprintf("http://localhost:5191/api/v1/File/%d", fileID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code when deleting file: %d", resp.StatusCode)
	}

	fmt.Println("File deleted successfully from server")
	return nil
}

// handleFileDeletion processes file deletions
func HandleFileDeletion(filePath string) {
	fileID, err := GetFileIDByName(filepath.Base(filePath))
	if err == nil {
		fmt.Println("Deleting file from server:", fileID)
		DeleteFileOnServer(fileID)
	} else {
		fmt.Println("Error finding file ID for deletion:", err)
	}
}

// UploadFileWithDebounce uploads a file with debouncing to prevent duplicate uploads
func UploadFileWithDebounce(filePath string) {
	uploadMutex.Lock() // Lock to prevent concurrent uploads
	defer uploadMutex.Unlock()

	// Wait for a short delay before proceeding to avoid rapid consecutive events
	time.Sleep(500 * time.Millisecond)

	// Check if the file still exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("File no longer exists, skipping upload:", filePath)
		return
	}

	// Check if the file was uploaded recently
	if lastTime, exists := lastUploaded[filePath]; exists {
		// Skip the upload if it was done within the last 2 seconds
		if time.Since(lastTime) < 2*time.Second {
			fmt.Println("Skipping duplicate upload:", filePath)
			return
		}
	}

	// Check if the file already exists on the server
	fileID, err := GetFileIDByName(filepath.Base(filePath))
	if err == nil && fileID > 0 {
		// If the file exists on the server, skip the upload
		fmt.Println("File already exists on the server, skipping upload:", filePath)
		return
	}

	// Upload the file if it's not a duplicate
	UploadFile(filePath)

	// Update the last upload time for the file
	lastUploaded[filePath] = time.Now()
}

// UploadFile uploads a new or modified file to the API using multipart/form-data
func UploadFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create a buffer and multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file
	part, err := writer.CreateFormFile("FileAttachment", filepath.Base(filePath))
	if err != nil {
		fmt.Println("Error creating form file:", err)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		fmt.Println("Error copying file to form part:", err)
		return
	}

	// Add the name field
	_ = writer.WriteField("Name", filepath.Base(filePath))

	// Close the writer to finalize the multipart form
	err = writer.Close()
	if err != nil {
		fmt.Println("Error closing writer:", err)
		return
	}

	// Create request
	req, err := http.NewRequest("POST", "http://localhost:5191/api/v1/File", body)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error uploading file:", err)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to upload file, status code:", resp.StatusCode)
		return
	}

	fmt.Println("File uploaded successfully:", filePath)
}
