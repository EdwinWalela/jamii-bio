package routes

import (
	"bytes"
	"crypto/md5"
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

	defer idFile.Close()
	defer faceFile.Close()

	// Send Images to Azure
	url := os.Getenv("AZUREBASEURL") + Attributes

	idPath = strings.Replace(idPath, "\\", "/", -1)
	idImgUrl := os.Getenv("SERVER_URL") + idPath

	jsonBody := fmt.Sprintf(`{"url":"%s"}`, idImgUrl)
	log.Println(jsonBody)

	body := []byte(jsonBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Ocp-Apim-Subscription-Key", os.Getenv("API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	resBody, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(resBody))
	// Return with Face ID(s)

	// Delete images

	_, _ = io.WriteString(w, "File uploaded")
	_, _ = io.Copy(idFile, idFace)
	_, _ = io.Copy(faceFile, userFace)

}

func VerificationHandler(w http.ResponseWriter, r *http.Request) {
	return
}
