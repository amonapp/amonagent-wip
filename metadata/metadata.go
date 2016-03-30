package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// GetURL - XXX
func GetURL(provider string, url string) string {
	req, RequestErr := http.NewRequest("GET", url, nil)
	if provider == "google" {
		req.Header.Set("Metadata-Flavor", "Google")
	}
	if RequestErr != nil {
		return ""
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 209 {
		return ""
	}

	data, bodyErr := ioutil.ReadAll(resp.Body)
	if bodyErr != nil {
		return ""
	}

	id := string(data)

	return id
}

func main() {
	MetadataURLs := map[string]string{
		"google":       "http://metadata.google.internal/computeMetadata/v1/instance/id",
		"amazon":       "http://169.254.169.254/latest/meta-data/instance-id",
		"digitalocean": "http://169.254.169.254/metadata/v1/id",
	}
	var CloudID string
	for provider, url := range MetadataURLs {
		response := GetURL(provider, url)
		if len(response) > 0 {
			CloudID = response
			break
		}
	}
	fmt.Println(CloudID)

}
