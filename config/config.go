package config

import (
	"fmt"
	"strings"
)

type Configuration struct {
	APIBaseURL string
	TCPURL     string
	Debug      bool
}

var Config Configuration

func Initialize() {
	Config = Configuration{
		APIBaseURL: "https://bytebridge.es8.nl/api/v1",
		TCPURL:     "bytebridge.es8.nl:8080",
		Debug:      false,
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
