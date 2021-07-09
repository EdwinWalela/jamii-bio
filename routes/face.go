package routes

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const UploadPath = "/static/images/"

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

	// Return with Face ID(s)

	// Delete images

	_, _ = io.WriteString(w, "File uploaded")
	_, _ = io.Copy(idFile, idFace)
	_, _ = io.Copy(faceFile, userFace)

}

func VerificationHandler(w http.ResponseWriter, r *http.Request) {
	return
}
