package main

import (
	"ByteBridge-Client/config"
	"ByteBridge-Client/socket"
	"ByteBridge-Client/sync"
	"ByteBridge-Client/watcher"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Initialize configuration
	config.Initialize()

	// Add command-line flags
	debug := flag.Bool("debug", false, "Enable debug logging")
	apiURL := flag.String("api", "", "Override API base URL (e.g., http://localhost:5191/api/v1)")
	flag.Parse()

	// Apply command line arguments to config
	if *debug {
		config.Config.Debug = true
		config.Config.APIBaseURL = "http://localhost:5191/api/v1"
		config.Config.TCPURL = "localhost:5000"
		fmt.Println("Debug mode enabled - using local API and TCP URL")
		fmt.Println("Debug mode enabled - verbose logging active")
	}

	if *apiURL != "" {
		config.SetAPIBaseURL(*apiURL)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error fetching home dir: ", err)
		return
	}

	syncFolder := filepath.Join(home, "Documents", "SyncFolder")
	// Ensure the sync folder exists
	if _, err := os.Stat(syncFolder); os.IsNotExist(err) {
		fmt.Println("Creating sync folder:", syncFolder)
		if err := os.MkdirAll(syncFolder, 0755); err != nil {
			fmt.Println("Error creating sync folder:", err)
			return
		}
	}

	// Channel to signal stopping the socket listener
	stopSocketChan := make(chan bool)

	// Startup Routine - initial sync
	go sync.CheckAndSyncFiles(syncFolder)

	// Start the folder watcher in a separate goroutine
	go watcher.WatchFolder(syncFolder)

	// Start the TCP socket listener - this will now trigger syncs based on server notifications
	go socket.ListenForServerChanges(syncFolder, stopSocketChan)

	fmt.Println("ByteBridge-Client started successfully")
	fmt.Printf("Syncing files with: %s\n", syncFolder)
	fmt.Printf("API endpoint: %s\n", config.Config.APIBaseURL)

	// Keep the program running
	for {
		time.Sleep(time.Hour)
	}
}
