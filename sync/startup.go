package sync

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// CheckAndSyncFiles checks and synchronizes files on startup
func CheckAndSyncFiles(syncFolder string) {
	log.Println("Starting file synchronization...")

	// Remove files that have been marked as deleted on the server
	deletedFiles, err := FetchDeletedFiles()
	if err != nil {
		log.Println("Error fetching deleted files from server:", err)
	} else {
		log.Println("Deleted files response:", deletedFiles)
		for _, file := range deletedFiles {
			filePath := filepath.Join(syncFolder, file.Name)
			if err := os.Remove(filePath); err == nil {
				log.Println("Deleted file:", filePath)
			} else {
				log.Println("Error deleting file:", filePath, err)
			}
		}
	}

	// Retrieve the list of files from the server
	files, err := FetchFiles()
	if err != nil {
		log.Println("Error fetching files from server:", err)
		return
	}
	log.Println("Files response:", files)

	// Create a map of local files and their hashes
	localFiles := make(map[string]string)
	dirEntries, err := os.ReadDir(syncFolder)
	if err == nil {
		for _, entry := range dirEntries {
			if !entry.IsDir() {
				filePath := filepath.Join(syncFolder, entry.Name())
				if hash, err := CalculateFileHash(filePath); err == nil {
					localFiles[entry.Name()] = hash
					log.Println("Found local file:", entry.Name(), "Hash:", hash)
				} else {
					log.Println("Error calculating hash for", entry.Name(), err)
				}
			}
		}
	} else {
		log.Println("Error reading sync folder:", err)
	}

	// Check if files are missing or modified and download them
	for _, serverFile := range files {
		localHash, exists := localFiles[serverFile.Name]
		if !exists || localHash != serverFile.Hash {
			log.Println("Downloading missing or changed file:", serverFile.Name)
			DownloadFile(syncFolder, serverFile.ID, serverFile.Name)
		} else {
			log.Println("File is up-to-date:", serverFile.Name)
		}
	}

	// Check if there are local files that are not on the server and upload them
	for localName := range localFiles {
		found := false
		for _, serverFile := range files {
			if localName == serverFile.Name {
				found = true
				break
			}
		}
		if !found {
			log.Println("Uploading new local file:", localName)
			UploadFile(syncFolder, filepath.Join(syncFolder, localName))
		}
	}

	log.Println("File synchronization completed.")
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

// CalculateFileHash computes the MD5 hash of a file, which is used to check if files are identical
func CalculateFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := fmt.Sprintf("%x", md5.Sum(data))
	return hash, nil
}
