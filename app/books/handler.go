package books

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type FuncType func(http.ResponseWriter, *http.Request)

var BookHandler FuncType = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	page := vars["page"]

	fmt.Fprintf(w, "You've requested the book: %s on page %s\n", title, page)
}

var BookHandlerStrict FuncType = func(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	vars := mux.Vars(r)
	title := vars["title"]

	fmt.Fprintf(w, "You've requested the book: %s from host: %s\n", title, host)
}

var BookRouter func(*mux.Router) = func(r *mux.Router) {
	bookrouter := r.PathPrefix("/books").Subrouter()

	bookrouter.HandleFunc("/{title}/page/{page}", BookHandler).Methods("GET")
	bookrouter.HandleFunc("/{title}", BookHandlerStrict).Host("localhost").Methods("GET")
}
