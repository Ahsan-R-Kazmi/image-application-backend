package main

import (
	"html/template"
	"log"
	"net/http"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "image-application"
)

func main() {
	templates := template.Must(template.ParseFiles("web/templates/index.html"))

	// Serve the /web/static directory containing css, images, etc. for the html template pages.
	http.Handle("/web/static/",
		http.StripPrefix("/web/static/",
			http.FileServer(http.Dir("web/static"))))

	// At the "/" path display the index page.
	http.HandleFunc("/" , func(w http.ResponseWriter, r *http.Request) {
		if e := templates.ExecuteTemplate(w, "index.html", nil); e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func CheckDbError(err error) {

}