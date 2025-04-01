package sync

// Contains logic for uploading files

import (
	"ByteBridge-Client/config"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Contains file upload logic

func UploadFileWithDebounce(syncFolder, filePath string) {
	uploadMutex.Lock()
	defer uploadMutex.Unlock()

	time.Sleep(500 * time.Millisecond)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		config.DebugLogger.Println("File no longer exists, skipping upload:", filePath)
		return
	}

	// Compute file hash
	fileHash, err := CalculateFileHash(filePath)
	if err != nil {
		config.DebugLogger.Println("Error computing file hash:", err)
		return
	}

	// Check if the file was uploaded recently
	if lastTime, exists := lastUploaded[filePath]; exists {
		if time.Since(lastTime) < 2*time.Second {
			config.DebugLogger.Println("Skipping duplicate upload:", filePath)
			return
		}
	}

	// Check if a file with the same hash already exists on the server
	fileID, err := GetFileIDByHash(fileHash)
	if err == nil && fileID > 0 {
		config.DebugLogger.Println("File with same hash exists on server, skipping upload:", filePath)
		return
	}

	UploadFile(syncFolder, filePath)
	lastUploaded[filePath] = time.Now()
}

func UploadFile(syncFolder, filePath string) {
	// Get the relative path of the file within the sync folder
	relativePath, err := filepath.Rel(syncFolder, filePath)
	if err != nil {
		config.DebugLogger.Println("Error getting relative path:", err)
		return
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		config.DebugLogger.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create a buffer and multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file to the form data
	part, err := writer.CreateFormFile("FileAttachment", filepath.Base(filePath))
	if err != nil {
		config.DebugLogger.Println("Error creating form file:", err)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		config.DebugLogger.Println("Error copying file to form part:", err)
		return
	}

	// Add additional fields
	if err := writer.WriteField("Name", relativePath); err != nil {
		config.DebugLogger.Println("Error writing 'Name' field:", err)
		return
	}
	config.DebugLogger.Println("Added form field: Name =", relativePath)

	// Properly close the writer
	err = writer.Close()
	if err != nil {
		config.DebugLogger.Println("Error closing writer:", err)
		return
	}

	// Log request preview (limited to 500 characters)
	config.DebugLogger.Println("Request Body (preview):", body.String())

	// Create the HTTP request
	req, err := http.NewRequest("POST", config.APIEndpoint("/File"), body)
	if err != nil {
		config.DebugLogger.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Log request details
	config.DebugLogger.Println("---- HTTP POST Request ----")
	config.DebugLogger.Println("URL:", req.URL)
	config.DebugLogger.Println("Headers:", req.Header)
	config.DebugLogger.Println("Headers:", req.Body)
	config.DebugLogger.Println("---------------------------")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		config.DebugLogger.Println("Error uploading file:", err)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		config.DebugLogger.Println("Failed to upload file, status code:", resp.StatusCode)
		return
	}

	config.DebugLogger.Println("File uploaded successfully:", filePath)
}
