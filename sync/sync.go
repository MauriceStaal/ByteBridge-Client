package sync

// Contains sync logic for checking and downloading files

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Global configuration
var (
	// BaseURL is the main API endpoint
	// BaseURL = "https://bytebridge.es8.nl"
	BaseURL = "http://localhost:5191"
	// APIEndpoint is the REST API path
	APIEndpoint = "/api/v1"
	// TCPHost is the host for the TCP notification service
	TCPHost = "localhost:5000"
)

// GetAPIURL returns the full API URL with endpoint
func GetAPIURL(path string) string {
	return fmt.Sprintf("%s%s%s", BaseURL, APIEndpoint, path)
}

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
	url := GetAPIURL("/File")
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

// SyncFiles establishes a TCP connection to the server and handles file sync events
func SyncFiles(syncFolder string) {
	// Initial sync to get all files at startup
	initialSync(syncFolder)

	// Setup reconnection with backoff
	backoffTime := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		fmt.Println("Connecting to file notification server...")
		// Connect to TCP server
		conn, err := net.Dial("tcp", TCPHost)
		if err != nil {
			fmt.Printf("TCP connection failed: %v\n", err)
			fmt.Printf("Retrying in %v seconds...\n", backoffTime.Seconds())
			time.Sleep(backoffTime)

			// Increase backoff time for next attempt, up to maximum
			backoffTime *= 2
			if backoffTime > maxBackoff {
				backoffTime = maxBackoff
			}
			continue
		}

		// Reset backoff on successful connection
		backoffTime = 1 * time.Second
		fmt.Println("TCP connection established")

		// Handle TCP connection
		handleTCPConnection(conn, syncFolder)

		// If we get here, the connection was closed
		fmt.Println("TCP connection closed, reconnecting...")
		conn.Close()
		time.Sleep(backoffTime)
	}
}

// initialSync performs a one-time sync of all files when the program starts
func initialSync(syncFolder string) {
	fmt.Println("Performing initial sync...")

	// Fetch the list of files from the server
	files, err := FetchFiles()
	if err != nil {
		fmt.Println("Error fetching files during initial sync:", err)
		return
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

	fmt.Println("Initial sync completed")
}

// handleTCPConnection processes messages from the TCP socket
func handleTCPConnection(conn net.Conn, syncFolder string) {
	// Send a hello message to the server to verify the connection is working
	fmt.Println("TCP connection established, sending hello message...")
	_, err := conn.Write([]byte("HELLO\n"))
	if err != nil {
		fmt.Println("Warning: Failed to send hello message:", err)
	}

	// Set a read deadline to detect connection issues
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		err = tcpConn.SetKeepAlive(true)
		if err != nil {
			fmt.Println("Warning: Failed to set keep alive:", err)
		}
		err = tcpConn.SetKeepAlivePeriod(30 * time.Second)
		if err != nil {
			fmt.Println("Warning: Failed to set keep alive period:", err)
		}
	}

	fmt.Println("Waiting for server messages...")

	// Create a buffer to read directly from the connection
	buffer := make([]byte, 1024)

	for {
		// Try direct reading from the connection first
		n, err := conn.Read(buffer)

		if err != nil {
			if err == io.EOF {
				fmt.Println("Server closed the connection")
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("Connection timed out")
			} else {
				fmt.Println("Error reading from TCP socket:", err)
			}
			break
		}

		if n > 0 {
			// Log the raw message for debugging
			rawMessage := string(buffer[:n])
			fmt.Printf("Raw message received (%d bytes): %q\n", n, rawMessage)

			// Process each line in the message
			lines := strings.Split(strings.ReplaceAll(rawMessage, "\r\n", "\n"), "\n")
			for _, line := range lines {
				// Skip empty lines
				if line == "" {
					continue
				}

				// Trim whitespace
				line = strings.TrimSpace(line)
				fmt.Printf("Processed line: %q\n", line)

				// Check if the message indicates files have changed
				if strings.Contains(line, "Files are changed") {
					fmt.Println("Server notification: Files have changed, syncing...")
					syncAllFiles(syncFolder)
				} else {
					fmt.Printf("Unrecognized message format: %q\n", line)
				}
			}
		} else {
			fmt.Println("Received 0 bytes, connection may be closed")
			time.Sleep(100 * time.Millisecond) // Small pause to avoid busy loop
		}
	}

	fmt.Println("TCP message handling loop exited")
}

// syncAllFiles fetches the current list of files and syncs them
func syncAllFiles(syncFolder string) {
	// Fetch the list of files from the server
	files, err := FetchFiles()
	if err != nil {
		fmt.Println("Error fetching files during sync:", err)
		return
	}

	// Track files that exist on the server to later detect deleted files
	serverFiles := make(map[string]bool)

	// Check for each file if it exists on the client, and if not, download it
	for _, file := range files {
		serverFiles[file.Name] = true
		localPath := filepath.Join(syncFolder, file.Name)

		if !FileExists(syncFolder, file.Name) {
			fmt.Println("File not found locally, downloading:", file.ID, file.Name)
			err := DownloadFile(syncFolder, file.ID, file.Name)
			if err != nil {
				fmt.Println("Error downloading file:", err)
			}
		} else {
			// File exists locally, we could check if it's different from the server version
			// using the hash field from the File struct
			localFileInfo, err := os.Stat(localPath)
			if err != nil {
				fmt.Println("Error checking local file:", err)
				continue
			}

			// Check if file size or modification time suggests we need to update
			// This is a simple heuristic - using file.Hash would be more accurate
			fmt.Printf("Checking if %s needs updating...\n", file.Name)

			// Parse the server's updated time
			serverTime, err := time.Parse(time.RFC3339, file.UpdatedOn)
			if err == nil && localFileInfo.ModTime().Before(serverTime) {
				fmt.Printf("Local file is older than server version, updating: %s\n", file.Name)
				err := DownloadFile(syncFolder, file.ID, file.Name)
				if err != nil {
					fmt.Println("Error updating file:", err)
				}
			} else {
				fmt.Printf("Local file %s is up to date\n", file.Name)
			}
		}
	}

	// Check for files that exist locally but not on the server (deleted files)
	localFiles, err := os.ReadDir(syncFolder)
	if err != nil {
		fmt.Println("Error reading sync folder:", err)
		return
	}

	for _, localFile := range localFiles {
		if !localFile.IsDir() {
			// If file exists locally but not on server, delete it
			if _, exists := serverFiles[localFile.Name()]; !exists {
				fmt.Println("File deleted on server, removing locally:", localFile.Name())
				err := os.Remove(filepath.Join(syncFolder, localFile.Name()))
				if err != nil {
					fmt.Println("Error deleting local file:", err)
				}
			}
		}
	}

	fmt.Println("Sync completed")
}

// DeleteFileOnServer deletes a file from the server
func DeleteFileOnServer(fileID int) error {
	url := GetAPIURL(fmt.Sprintf("/File/%d", fileID))
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
	req, err := http.NewRequest("POST", GetAPIURL("/File"), body)
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
