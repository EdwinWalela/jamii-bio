package routes

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
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
	MissingFace  []string `json:"missing-face"`
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

type VerificationBody struct {
	Face1 string `json:"face1"`
	Face2 string `json:"face2"`
}

type AzureVerificationRes struct {
	IsIdentical bool    `json:"isIdentical"`
	Confidence  float64 `json:"confidence"`
}

type VerifyFaceResponse struct {
	Match bool `json:"match"`
}

func hashFileName(filename string) string {
	h := md5.New()
	h.Write([]byte(filename))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func cropImage(img multipart.File) (image.Image, image.Image, error) {
	pic, err := jpeg.Decode(img)
	if err != nil {
		log.Println(err)
	}

	croppedImg := imaging.CropAnchor(pic, 600, 800, imaging.TopLeft)
	croppedImg = imaging.Rotate90(croppedImg)
	croppedImg = imaging.AdjustBrightness(croppedImg, 10)

	pic = imaging.Rotate90(pic)

	if err != nil {
		return nil, nil, err
	}
	return croppedImg, pic, nil
}

func DetectHandler(w http.ResponseWriter, r *http.Request) {
	idFace, idFormFile, idErr := r.FormFile("id")
	userFace, faceFormFile, faceErr := r.FormFile("face")
	idDetails := idFace
	idDetailsFormFile := idFormFile

	res := &DetectFaceResponse{
		EmotionMatch: false,
		FaceId:       []string{},
		MissingFace:  []string{},
	}

	if idFormFile == nil {
		res.MissingFace = append(res.MissingFace, "id")
	}

	if faceFormFile == nil {
		res.MissingFace = append(res.MissingFace, "selfie")
	}

	// Return if one or both file(s) are missing
	if idFormFile == nil || faceFormFile == nil {
		json.NewEncoder(w).Encode(res)
		return
	}

	idFaceFileName := hashFileName(idFormFile.Filename) + ".jpg"
	userFaceFileName := hashFileName(faceFormFile.Filename) + ".jpg"
	idDetailsFileName := hashFileName(idDetailsFormFile.Filename) + "-ocr.jpg"

	if idErr != nil {
		panic(idErr)
	}
	if faceErr != nil {
		panic(faceErr)
	}

	defer idFace.Close()
	defer userFace.Close()
	defer idDetails.Close()

	idPath := filepath.Join(".", UploadPath, idFaceFileName)
	facePath := filepath.Join(".", UploadPath, userFaceFileName)
	idDetailsPath := filepath.Join(".", UploadPath, idDetailsFileName)

	idFile, err := os.OpenFile(idPath, os.O_WRONLY|os.O_CREATE, 0666)
	faceFile, err := os.OpenFile(facePath, os.O_WRONLY|os.O_CREATE, 0666)
	idDetailsFile, err := os.OpenFile(idDetailsPath, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		panic(err)
	}
	croppedImg, originalImg, err := cropImage(idFace)

	if err != nil {
		log.Println(err)
	}

	buffCropped := bytes.NewBuffer([]byte{})
	buffOriginal := bytes.NewBuffer([]byte{})

	jpeg.Encode(buffCropped, croppedImg, &jpeg.Options{Quality: 100})
	jpeg.Encode(buffOriginal, originalImg, &jpeg.Options{Quality: 100})

	_, _ = io.Copy(idFile, buffCropped)
	_, _ = io.Copy(faceFile, userFace)
	_, _ = io.Copy(idDetailsFile, buffOriginal)

	// Send Images to Azure
	reqUrl := os.Getenv("AZURE_DETECT_BASEURL") + Attributes

	idNewPath := strings.Replace(idPath, "\\", "/", -1)
	idImgUrl := os.Getenv("SERVER_URL") + idNewPath

	faceNewPath := strings.Replace(facePath, "\\", "/", -1)
	faceUrl := os.Getenv("SERVER_URL") + faceNewPath

	faceUrls := []string{idImgUrl, faceUrl}
	var detectedFaces []DetectedFace

	for i, url := range faceUrls {
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
			if i == 0 {
				res.MissingFace = append(res.MissingFace, "id")
			}
			if i == 1 {
				res.MissingFace = append(res.MissingFace, "selfie")
			}
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

	// Return with Face ID(s)
	for _, face := range detectedFaces {
		// if face.Suprise > 0.4 { // check if emotion match
		res.EmotionMatch = true
		// }
		res.FaceId = append(res.FaceId, face.FaceId)
	}

	if !res.EmotionMatch {
		res.FaceId = []string{}
	}

	// Delete images
	idFile.Close()
	faceFile.Close()
	idDetailsFile.Close()

	// if e := os.Remove(idPath); e != nil {
	// 	log.Println(e)
	// }
	// if e := os.Remove(facePath); e != nil {
	// 	log.Println(e)
	// }

	json.NewEncoder(w).Encode(res)
	return
}

func VerificationHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)

	var verBody VerificationBody

	// parse json request body (face1 & face2)
	if err := decoder.Decode(&verBody); err != nil {
		log.Println(err)
		return
	}

	// Pack urls to json payload
	jsonBody := fmt.Sprintf(`{"faceId1":"%s","faceId2":"%s"}`, verBody.Face1, verBody.Face2)
	fmt.Println(jsonBody)
	reqBody := []byte(jsonBody)

	reqURL := os.Getenv("AZURE_VERIFY_BASEURL")

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(reqBody))

	// Set Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", os.Getenv("API_KEY"))

	client := &http.Client{}

	// Send request
	resp, err := client.Do(req)

	if err != nil {

		log.Println(err)
		return
	}

	defer resp.Body.Close()

	var res VerifyFaceResponse

	res.Match = false

	// Serialize Azure response
	resBody, _ := ioutil.ReadAll(resp.Body)

	var parsedJson AzureVerificationRes

	if err := json.Unmarshal(resBody, &parsedJson); err != nil {

		log.Println(err)
		return
	}

	if parsedJson.IsIdentical == true || parsedJson.Confidence > 0.40 {
		res.Match = true
	}

	json.NewEncoder(w).Encode(res)
	return

}
