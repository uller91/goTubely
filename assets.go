package main

import (
	"os"
	"strings"
	"crypto/rand"
	"encoding/base64"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}


func getFileType(mimeType string) string {
	split := strings.Split(mimeType, "/")
	var fileType string
	if len(split) > 1 {
    	fileType = split[1]
	} else {
	    fileType = split[0]
	}
	return fileType
}


func getRandomKey() string {
	key := make([]byte, 32)
	rand.Read(key)
	return base64.URLEncoding.EncodeToString(key)
}