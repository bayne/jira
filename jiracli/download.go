package jiracli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DownloadToFile marshals data as JSON and writes it to a file.
func DownloadToFile(filename string, data interface{}) error {
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	jsonFile := filename + ".json"
	if err := os.WriteFile(jsonFile, buf, 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Downloaded: %s\n", jsonFile)
	return nil
}

// DownloadToDir creates the given directory and writes each item as JSON.
// The keyFunc extracts the filename from each item.
func DownloadToDir(dir string, items []interface{}, keyFunc func(interface{}) string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, item := range items {
		key := keyFunc(item)
		filename := filepath.Join(dir, key)
		if err := DownloadToFile(filename, item); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "Downloaded %d items to %s/\n", len(items), dir)
	return nil
}
