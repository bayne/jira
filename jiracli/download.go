package jiracli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// DownloadToFile renders the given template with data and writes the result to a file.
func DownloadToFile(filename, templateName string, data interface{}) error {
	var buf bytes.Buffer
	if err := RunTemplate(templateName, data, &buf); err != nil {
		return err
	}
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Downloaded: %s\n", filename)
	return nil
}

// DownloadToDir creates the given directory and renders each item using the
// template, writing one file per item. The keyFunc extracts the filename from each item.
func DownloadToDir(dir, templateName string, items []interface{}, keyFunc func(interface{}) string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, item := range items {
		key := keyFunc(item)
		filename := filepath.Join(dir, key)
		if err := DownloadToFile(filename, templateName, item); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "Downloaded %d items to %s/\n", len(items), dir)
	return nil
}
