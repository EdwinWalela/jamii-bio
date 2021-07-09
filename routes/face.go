package routes

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const UploadPath = "/static/images/"

func DetectHandler(w http.ResponseWriter, r *http.Request) {
	idFace, idFormFile, idErr := r.FormFile("id")
	userFace, faceFormFile, faceErr := r.FormFile("face")

	idFaceFileName := "id-1234." + strings.Split(idFormFile.Filename, ".")[1]
	userFaceFileName := "face-1234." + strings.Split(faceFormFile.Filename, ".")[1]

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

	_, _ = io.WriteString(w, "File uploaded")
	_, _ = io.Copy(idFile, idFace)

}

func VerificationHandler(w http.ResponseWriter, r *http.Request) {
	return
}
