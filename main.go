package main

import (
	"ByteBridge-Client/sync"
	"ByteBridge-Client/watcher"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error fetching home dir: ", err)
		return
	}

	syncFolder := filepath.Join(home, "Documents", "SyncFolder")
	// Start the folder watcher in a separate goroutine
	go watcher.WatchFolder(syncFolder)

	// Start syncing files in another goroutine
	go sync.SyncFiles(syncFolder)

	time.Sleep(time.Hour * 24) // Keep the watcher running
}
