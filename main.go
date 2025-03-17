package main

import (
	"ByteBridge-Client/sync"
	"ByteBridge-Client/watcher"
	"time"
)

const syncFolder = "/home/erwin/Documents/ByteBridge-Client"

func main() {
	// Start the folder watcher in a separate goroutine
	go watcher.WatchFolder(syncFolder)

	// Start syncing files in another goroutine
	go sync.SyncFiles(syncFolder)

	time.Sleep(time.Hour * 24) // Keep the watcher running
}
