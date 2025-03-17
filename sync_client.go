package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
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

const syncFolder = "/home/erwin/Documents/GoSync"

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
func FileExists(filename string) bool {
	filePath := filepath.Join(syncFolder, filename)
	_, err := os.Stat(filePath)
	return err == nil
}

// DownloadFile downloads a missing file from the API
func DownloadFile(fileID int, filename string) error {
	url := fmt.Sprintf("http://localhost:5191/api/v1/File/%d", fileID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file %s: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code when downloading %s: %d", filename, resp.StatusCode)
	}

	filePath := filepath.Join(syncFolder, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file %s: %w", filename, err)
	}

	return nil
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

// WatchFolder watches for changes in the sync folder and uploads new, modified, or deleted files
func WatchFolder() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
					fmt.Println("Detected change in:", event.Name)
					UploadFile(event.Name)
				} else if event.Op&fsnotify.Remove != 0 {
					fmt.Println("Detected deletion of:", event.Name)
					fileID, err := GetFileIDByName(filepath.Base(event.Name))
					if err == nil {
						DeleteFileOnServer(fileID)
					} else {
						fmt.Println("Error finding file ID for deletion:", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("Watcher error:", err)
			}
		}
	}()

	if err := watcher.Add(syncFolder); err != nil {
		fmt.Println("Error adding folder to watcher:", err)
		return
	}
	<-done
}

// UploadFile uploads a new or modified file to the API
func UploadFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"Name":           filepath.Base(filePath),
		"FileAttachment": fileBytes,
	})
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	resp, err := http.Post("http://localhost:5191/api/v1/File", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error uploading file:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to upload file, status code:", resp.StatusCode)
		return
	}

	fmt.Println("File uploaded successfully:", filePath)
}

func main() {
	go WatchFolder()
	time.Sleep(time.Hour * 24) // Keep the watcher running
}
