package sync

// Contains logic for deleting files

import (
	"ByteBridge-Client/config"
	"fmt"
	"net/http"
	"path/filepath"
)

// DeleteFileOnServer deletes a file from the server
func DeleteFileOnServer(fileID int) error {
	url := fmt.Sprintf(config.APIEndpoint("/File/%d"), fileID)
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
