package watcher

import (
	"ByteBridge-Client/sync"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
)

// WatchFolder watches for changes in the sync folder and uploads new, modified, or deleted files
// It also watches subfolders continuously, including new subfolder creation.
func WatchFolder(syncFolder string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	// Initial folder scan to add existing folders and subfolders
	err = watchSubfolders(syncFolder, watcher)
	if err != nil {
		fmt.Println("Error while watching subfolders:", err)
		return
	}

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
					sync.UploadFileWithDebounce(syncFolder, event.Name)

					// If a new subfolder is created, add it to the watcher
					if isDirectory(event.Name) {
						err := watcher.Add(event.Name)
						if err != nil {
							fmt.Println("Error adding new subfolder to watcher:", err)
						} else {
							fmt.Println("Started watching new subfolder:", event.Name)
						}
					}

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

	<-done
}

// watchSubfolders recursively adds subfolders to the watcher
func watchSubfolders(syncFolder string, watcher *fsnotify.Watcher) error {
	err := filepath.Walk(syncFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only add directories to the watcher
		if info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				fmt.Println("Error adding folder to watcher:", err)
				return err
			}
			fmt.Println("Watching folder:", path)
		}
		return nil
	})

	return err
}

// isDirectory checks if the given path is a directory
func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}
