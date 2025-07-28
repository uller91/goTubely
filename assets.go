package main

import (
	"os"
	"os/exec"
	"bytes"
	"encoding/json"
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


func processVideoForFastStart(filePath string) (string, error) {
	newPath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return newPath, nil
}


func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	type Output struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
	}

	var output Output
	err = json.Unmarshal(out.Bytes(), &output)
	if err != nil {
        return "", err
    }

	if output.Streams[0].Width * 9 / output.Streams[0].Height == 16 {
		return "landscape", nil
	} else if output.Streams[0].Width * 16 / output.Streams[0].Height == 9 {
		return "portrait", nil
	} else {
		return "other", nil
	}
}