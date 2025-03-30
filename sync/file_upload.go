package sync

// Contains logic for uploading files

import (
	"ByteBridge-Client/config"
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Contains file upload logic

// UploadFileWithDebounce uploads a file with debouncing to prevent duplicate uploads
func UploadFileWithDebounce(syncFolder, filePath string) {
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
	UploadFile(syncFolder, filePath)

	// Update the last upload time for the file
	lastUploaded[filePath] = time.Now()
}

// UploadFile uploads a new or modified file to the API using multipart/form-data
func UploadFile(syncFolder, filePath string) {
	// Get the relative path of the file within the sync folder
	relativePath, err := filepath.Rel(syncFolder, filePath)
	if err != nil {
		log.Println("Error getting relative path:", err)
		return
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create a buffer and multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file
	part, err := writer.CreateFormFile("FileAttachment", filepath.Base(filePath))
	if err != nil {
		log.Println("Error creating form file:", err)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		log.Println("Error copying file to form part:", err)
		return
	}

	// Add the name field with the relative path
	if err := writer.WriteField("Name", relativePath); err != nil {
		log.Println("Error writing 'Name' field:", err)
		return
	}

	// Add the path field (absolute path)
	if err := writer.WriteField("Path", filePath); err != nil {
		log.Println("Error writing 'Path' field:", err)
		return
	}

	// Log the fields before sending the request
	log.Println("Sending request with Name (relative path):", relativePath)
	log.Println("Sending request with Path (absolute path):", filePath)

	// Close the writer to finalize the multipart form
	err = writer.Close()
	if err != nil {
		log.Println("Error closing writer:", err)
		return
	}

	// Create request
	req, err := http.NewRequest("POST", config.APIEndpoint("/File"), body)
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Log the request headers
	log.Println("Making POST request to URL:", config.APIEndpoint("/File"))
	log.Println("Request Headers:", req.Header)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error uploading file:", err)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		log.Println("Failed to upload file, status code:", resp.StatusCode)
		return
	}

	log.Println("File uploaded successfully:", filePath)
}
