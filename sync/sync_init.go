package sync

// Contains sync logic that syncs everything on client startup

import (
	"log"
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
