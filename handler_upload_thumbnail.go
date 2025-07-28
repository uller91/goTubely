package main

import (
	"fmt"
	"net/http"
	"io"
	"path/filepath"
	"os"
	"mime"
	
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
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

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Improper mime type", err)
		return
	}
	
	fileType := getFileType(mimeType)
	UrlBase64 := getRandomKey()

	filePath := fmt.Sprintf("%v.%v", UrlBase64, fileType)
	fullFilePath := filepath.Join(cfg.assetsRoot, filePath)

	newFile, err := os.Create(fullFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file to store the thumbnail", err)
		return
	}

	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload the thumbnail to the file server", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You can't add thumbnail to this video", err)
		return
	}

	//fileThumbnail := thumbnail{data: fileData, mediaType: fileType}
	//videoThumbnails[videoID] = fileThumbnail

	fileThumbnailUrl := fmt.Sprintf("http://localhost:8091/assets/%v.%v", UrlBase64, fileType)
	video.ThumbnailURL = &fileThumbnailUrl
	//video.ThumbnailURL = &fileDataUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update the video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
