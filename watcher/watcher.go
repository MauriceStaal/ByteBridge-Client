package watcher

// Contains file watcher logic

import (
	"ByteBridge-Client/sync"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
)

// WatchFolder watches for changes in the sync folder and uploads new, modified, or deleted files
func WatchFolder(syncFolder string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				switch {
				case event.Op&(fsnotify.Create|fsnotify.Write) != 0:
					fmt.Println("Detected change in:", event.Name)
					sync.UploadFileWithDebounce(event.Name)

				case event.Op&fsnotify.Remove != 0:
					fmt.Println("Detected deletion of:", event.Name)
					sync.HandleFileDeletion(event.Name)

				case event.Op&fsnotify.Rename != 0:
					// Rename could mean either a rename or deletion (on Linux)
					if _, err := os.Stat(event.Name); os.IsNotExist(err) {
						fmt.Println("Detected possible deletion (rename event):", event.Name)
						sync.HandleFileDeletion(event.Name)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("Watcher error:", err)
			}
		}
	}()

	if err := watcher.Add(syncFolder); err != nil {
		fmt.Println("Error adding folder to watcher:", err)
		return
	}
	<-done
}
