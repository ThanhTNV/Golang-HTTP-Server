package main

import (
	"fmt"
	"net/http"

	"helloworld/app/books"
	"helloworld/app/pets"
	"helloworld/db"

	"github.com/gorilla/mux"
)

var port string = ":80"

func main() {
	// Initialize database connection
	db.InitDB()

	r := mux.NewRouter()
	fs := http.FileServer(http.Dir("static/"))

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	books.BookRouter(r)
	pets.SetupRoutes(r, db.DB)

	fmt.Printf("Server is running at: http://localhost%s\n", port)
	http.ListenAndServe(port, r)
}
