package main

import (
	"fmt"
	"net/http"
	"io"
	"encoding/base64"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	fileType := header.Header.Get("Content-Type")

	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read the file", err)
	}

	//working here
	fileDataBase64 := base64.StdEncoding.EncodeToString(fileData)
	fileDataUrl := fmt.Sprintf("data:%v;base64,%v", fileType, fileDataBase64)

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

	//fileThumbnailUrl := fmt.Sprintf("http://localhost:8091/api/thumbnails/%v", videoID)
	//video.ThumbnailURL = &fileThumbnailUrl
	video.ThumbnailURL = &fileDataUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update the video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
