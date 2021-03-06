package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/edwinwalela/jamii-bio/routes"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const PORT = 8000

var URL = fmt.Sprintf("0.0.0.0:%d", PORT)

func main() {
	godotenv.Load()
	r := mux.NewRouter()

	r.HandleFunc("/detect", routes.DetectHandler).Methods("POST")
	r.HandleFunc("/verify", routes.VerificationHandler).Methods("POST")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("static"))))

	srv := &http.Server{
		Addr:         URL,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	log.Printf("Listening for requests on port:%d\n", PORT)
	log.Fatal(srv.ListenAndServe())

}
