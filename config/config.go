package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// type Apache struct {
// }

// Mongo - XXX
type Mongo struct {
	URI string `json:"uri"`
}

// Config - XXX
type Config struct {
	Path string
	Name string
}

// ConfigPath - XXX
const ConfigPath = "/etc/amonagent/plugins-enabled"

// GetAllEnabledPlugins - XXX
func GetAllEnabledPlugins() ([]Config, error) {
	fileList := []Config{}
	filepath.Walk(ConfigPath, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			// Only files ending with .conf
			fileName := strings.Split(f.Name(), ".conf")
			if len(fileName) == 2 {
				f := Config{Path: path, Name: fileName[0]}
				fileList = append(fileList, f)
			}

		}
		return nil
	})

	return fileList, nil
}

func main() {
	EnabledPlugins, _ := GetAllEnabledPlugins()
	for _, p := range EnabledPlugins {
		fmt.Println(p)
	}

}
