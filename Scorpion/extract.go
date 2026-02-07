package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

func getFileInfo(pathOfFile string) map[string]string {
	info, err := os.Stat(pathOfFile)
	if err != nil {
		return map[string]string{}
	}

	tags := map[string]string{}
	tags["FileSize"] = fmt.Sprintf("%.0f kB", float64(info.Size())/1024)
	tags["FileModifyDate"] = info.ModTime().Format("2006:01:02 15:04:05+0000")
	tags["FileAccessDate"] = time.Now().Format("2006:01:02 15:04:05+0000")
	tags["FilePermissions"] = info.Mode().String()
	return tags
}

func openAndExtractContent(pathOfFile string) ([]byte, error) {
	file, err := os.Open(pathOfFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
		}
	}()

	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return b, nil
}
