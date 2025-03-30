package sync

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Contains file download logic

// DownloadFile downloads a missing file from the API
func DownloadFile(syncFolder string, fileID int, filename string) error {
	url := fmt.Sprintf("https://bytebridge.es8.nl/api/v1/File/%d", fileID)
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
