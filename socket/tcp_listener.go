package socket

import (
	"net"
	"sync"
	"time"

	// Import your sync package to call the CheckAndSyncFiles function
	// Make sure to use the correct import path based on your module name
	"ByteBridge-Client/config"
	syncPkg "ByteBridge-Client/sync"
)

// ListenForServerChanges listens for TCP messages from the server and
// triggers file synchronization when changes are detected
func ListenForServerChanges(syncFolder string, stopChan chan bool) {
	config.DebugLogger.Println("Starting TCP server change listener...")

	// Prevent multiple syncs happening at the same time
	var syncMutex sync.Mutex
	var syncing bool

	// Function to handle synchronization with debouncing
	triggerSync := func() {
		syncMutex.Lock()
		defer syncMutex.Unlock()

		if syncing {
			config.DebugLogger.Println("Sync already in progress, skipping this request")
			return
		}

		syncing = true
		go func() {
			config.DebugLogger.Println("Triggered sync based on server notification")
			syncPkg.CheckAndSyncFiles(syncFolder)

			syncMutex.Lock()
			syncing = false
			syncMutex.Unlock()
		}()
	}

	// Function to connect and listen for changes
	var connect func()
	connect = func() {
		config.DebugLogger.Printf("Connecting to TCP server at %s", config.Config.TCPURL)

		conn, err := net.Dial("tcp", config.Config.TCPURL)
		if err != nil {
			config.DebugLogger.Printf("Failed to connect to TCP server: %v. Retrying in 5 seconds...\n", err)
			time.Sleep(5 * time.Second)
			go connect()
			return
		}

		config.DebugLogger.Println("Successfully connected to TCP server")
		defer conn.Close()

		// Create a buffer to read incoming messages
		buffer := make([]byte, 1024)

		for {
			// Read message from server and place into buffer
			n, err := conn.Read(buffer)
			if err != nil {
				config.DebugLogger.Printf("Error reading from TCP server: %v. Reconnecting...", err)
				time.Sleep(3 * time.Second)
				go connect()
				return
			}

			message := string(buffer[:n])
			config.DebugLogger.Printf("Received message from server: %s", message)

			// Any message from the server indicates changes that require sync
			triggerSync()
		}
	}

	// Start the connection
	go connect()

	// Wait for stop signal
	<-stopChan
	config.DebugLogger.Println("TCP listener stopped")
}
