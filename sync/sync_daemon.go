package sync

// Contains sync logic

import (
	"fmt"
	"time"
)

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
