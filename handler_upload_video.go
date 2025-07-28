package main

import (
	"net/http"
	"fmt"
	"io"
	"os"
	"mime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxSize = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You can't add thumbnail to this video", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")	//extracts mime "type/subtype"
	mimeType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Improper Content-Type", err)
		return
	}

	if mimeType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Improper mime type", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file for the upload", err)
	}
	defer os.Remove(tempFile.Name()) // clean up
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write file into the temp file", err)
		return
	}

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset the temp file pointer", err)
		return
	}

	fileType := getFileType(mimeType)
	UrlBase64 := getRandomKey()

	tmpPath := fmt.Sprintf("%v", tempFile.Name())
	newPath, err := processVideoForFastStart(tmpPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process the video", err)
		return
	}
	processedFile, err := os.Open(newPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open the processed video", err)
		return
	}
	defer processedFile.Close()

	ratio, err := getVideoAspectRatio(newPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't define video's ratio", err)
		return
	}

	keyPath := fmt.Sprintf("%v/%v.%v", ratio, UrlBase64, fileType)
	params := &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket), //aws.String returns pointer to the string
		Key:         aws.String(keyPath),
		Body:        processedFile,
		ContentType: aws.String(mimeType),
	}
	_, err = cfg.s3Client.PutObject(r.Context(), params)

	fileVideoUrl := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, keyPath)
	video.VideoURL = &fileVideoUrl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update the video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
