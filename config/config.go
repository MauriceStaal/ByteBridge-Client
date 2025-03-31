package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type Configuration struct {
	APIBaseURL string
	TCPURL     string
	Debug      bool
}

var Config Configuration

var DebugLogger *log.Logger
var InfoLogger *log.Logger

func Initialize() {
	Config = Configuration{
		APIBaseURL: "https://bytebridge.es8.nl/api/v1",
		TCPURL:     "bytebridge.es8.nl:50500",
		Debug:      false,
	}

	InfoLogger = log.New(os.Stdout, "INFO: ", log.LstdFlags)

	DebugLogger = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
}

func SetDebugMode(enabled bool) {
	Config.Debug = enabled

	if enabled {
		// Enable debug logging to stdout
		DebugLogger.SetOutput(os.Stdout)
		DebugLogger.Println("Debug logging enabled")
	} else {
		// Disable debug logging
		DebugLogger.SetOutput(io.Discard)
	}
}

// APIEndpoint returns a full API endpoint URL for a given path
func APIEndpoint(path string) string {
	// Ensure path starts with a slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return Config.APIBaseURL + path
}

// SetAPIBaseURL updates the API base URL
func SetAPIBaseURL(url string) {
	Config.APIBaseURL = strings.TrimRight(url, "/")
	fmt.Println("API URL set to:", Config.APIBaseURL)
}

// SetTCPURL updates the TCP URL
func SetTCPURL(url string) {
	Config.TCPURL = strings.TrimRight(url, "/")
	fmt.Println("TCP URL set to:", Config.TCPURL)
}
