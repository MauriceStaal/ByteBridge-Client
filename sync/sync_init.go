package sync

// Contains sync logic that syncs everything on client startup

import (
	"ByteBridge-Client/config"
	"os"
	"path/filepath"
)

// CheckAndSyncFiles checks and synchronizes files on startup
func CheckAndSyncFiles(syncFolder string) {
	config.InfoLogger.Println("Starting file synchronization...")

	// Remove files that have been marked as deleted on the server
	deletedFiles, err := FetchDeletedFiles()
	if err != nil {
		config.InfoLogger.Println("Error fetching deleted files from server:", err)
	} else {
		config.DebugLogger.Println("Deleted files response:", deletedFiles)
		for _, file := range deletedFiles {
			filePath := filepath.Join(syncFolder, file.Name)
			if err := os.Remove(filePath); err == nil {
				config.InfoLogger.Println("Deleted file:", file.Name)
				config.DebugLogger.Println("Deleted file path:", filePath)
			} else if !os.IsNotExist(err) {
				config.InfoLogger.Println("Error deleting file:", file.Name, err)
			}
		}
	}

	// Retrieve the list of files from the server
	files, err := FetchFiles()
	if err != nil {
		config.InfoLogger.Println("Error fetching files from server:", err)
		return
	}
	config.DebugLogger.Println("Files response:", files)

	// Create a map of local files and their hashes
	localFiles := make(map[string]string)
	dirEntries, err := os.ReadDir(syncFolder)
	if err == nil {
		for _, entry := range dirEntries {
			if !entry.IsDir() {
				filePath := filepath.Join(syncFolder, entry.Name())
				if hash, err := CalculateFileHash(filePath); err == nil {
					localFiles[entry.Name()] = hash
					config.DebugLogger.Println("Found local file:", entry.Name(), "Hash:", hash)
				} else {
					config.DebugLogger.Println("Error calculating hash for", entry.Name(), err)
				}
			}
		}
	} else {
		config.InfoLogger.Println("Error reading sync folder:", err)
	}

	// Check if files are missing or modified and download them
	for _, serverFile := range files {
		localHash, exists := localFiles[serverFile.Name]
		if !exists || localHash != serverFile.Hash {
			config.InfoLogger.Println("Downloading file:", serverFile.Name)
			DownloadFile(syncFolder, serverFile.ID, serverFile.Name)
		} else {
			config.DebugLogger.Println("File is up-to-date:", serverFile.Name)
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
			config.InfoLogger.Println("Uploading new file:", localName)
			UploadFile(syncFolder, filepath.Join(syncFolder, localName))
		}
	}

	config.InfoLogger.Println("File synchronization completed.")
}
