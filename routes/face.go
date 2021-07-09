package routes

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const UploadPath = "/static/images/"

func DetectHandler(w http.ResponseWriter, r *http.Request) {
	idFace, idFaceHandler, idErr := r.FormFile("id")
	// userFace, userFaceHandler, faceErr := r.FormFile("face")

	if idErr != nil {
		panic(idErr)
	}
	// if faceErr != nil {
	// 	panic(faceErr)
	// }

	defer idFace.Close()
	// defer userFace.Close()
	path := filepath.Join(".", UploadPath)
	idFile, err := os.OpenFile(path+idFaceHandler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}

	defer idFile.Close()

	_, _ = io.WriteString(w, "File uploaded")
	_, _ = io.Copy(idFile, idFace)

}

func VerificationHandler(w http.ResponseWriter, r *http.Request) {
	return
}
