package routes

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const UploadPath = "/static/images/"

const Attributes = "?returnFaceAttributes=emotion,glasses"

type DetectedFace struct {
	FaceId    string
	Glasses   bool
	Suprise   float32
	Happiness float32
}

type DetectFaceResponse struct {
	EmotionMatch bool     `json:"emotion-match"`
	FaceId       []string `json:"face-id"`
}

type AzureResponse struct {
	FaceId         string
	FaceRectangle  interface{}
	FaceAttributes AzureFaceAttributes
}

type AzureFaceAttributes struct {
	Glasses string
	Emotion map[string]float32
}

func hashFileName(filename string) string {
	h := md5.New()
	h.Write([]byte(filename))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func DetectHandler(w http.ResponseWriter, r *http.Request) {
	idFace, idFormFile, idErr := r.FormFile("id")
	userFace, faceFormFile, faceErr := r.FormFile("face")

	idFaceFileName := hashFileName(idFormFile.Filename) + "." + strings.Split(idFormFile.Filename, ".")[1]
	userFaceFileName := hashFileName(faceFormFile.Filename) + "." + strings.Split(faceFormFile.Filename, ".")[1]

	if idErr != nil {
		panic(idErr)
	}
	if faceErr != nil {
		panic(faceErr)
	}

	defer idFace.Close()
	defer userFace.Close()

	idPath := filepath.Join(".", UploadPath, idFaceFileName)
	facePath := filepath.Join(".", UploadPath, userFaceFileName)

	idFile, err := os.OpenFile(idPath, os.O_WRONLY|os.O_CREATE, 0666)
	faceFile, err := os.OpenFile(facePath, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		panic(err)
	}

	_, _ = io.Copy(idFile, idFace)
	_, _ = io.Copy(faceFile, userFace)
	// Send Images to Azure
	reqUrl := os.Getenv("AZUREBASEURL") + Attributes

	idNewPath := strings.Replace(idPath, "\\", "/", -1)
	idImgUrl := os.Getenv("SERVER_URL") + idNewPath

	faceNewPath := strings.Replace(facePath, "\\", "/", -1)
	faceUrl := os.Getenv("SERVER_URL") + faceNewPath

	faceUrls := []string{idImgUrl, faceUrl}
	var detectedFaces []DetectedFace

	for _, url := range faceUrls {
		jsonBody := fmt.Sprintf(`{"url":"%s"}`, url)

		body := []byte(jsonBody)

		req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(body))

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Ocp-Apim-Subscription-Key", os.Getenv("API_KEY"))

		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			log.Println(err)
			break
		}

		defer resp.Body.Close()

		resBody, _ := ioutil.ReadAll(resp.Body)
		var parsedJsonMap []AzureResponse

		if err := json.Unmarshal(resBody, &parsedJsonMap); err != nil {
			log.Println(err)
			continue
		}

		if len(parsedJsonMap) == 0 {
			log.Println("No faces found")
			break
		}

		parsedFace := parsedJsonMap[0]
		detectedFace := DetectedFace{
			FaceId:    parsedFace.FaceId,
			Glasses:   parsedFace.FaceAttributes.Glasses == "NoGlasses",
			Suprise:   parsedFace.FaceAttributes.Emotion["surprise"],
			Happiness: parsedFace.FaceAttributes.Emotion["happiness"],
		}

		detectedFaces = append(detectedFaces, detectedFace)
	}

	res := &DetectFaceResponse{
		EmotionMatch: false,
		FaceId:       []string{},
	}
	// Return with Face ID(s)
	for _, face := range detectedFaces {
		if face.Suprise > 0.4 { // check if emotion match
			res.EmotionMatch = true
		}
		res.FaceId = append(res.FaceId, face.FaceId)
	}

	if !res.EmotionMatch {
		res.FaceId = []string{}
	}

	// Delete images
	idFile.Close()
	faceFile.Close()

	if e := os.Remove(idPath); e != nil {
		log.Println(e)
	}
	if e := os.Remove(facePath); e != nil {
		log.Println(e)
	}

	json.NewEncoder(w).Encode(res)

}

func VerificationHandler(w http.ResponseWriter, r *http.Request) {
	return
}
