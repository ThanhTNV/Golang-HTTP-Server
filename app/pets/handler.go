package pets

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Pet struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name"`
	Type string `json:"type"`
	Age  int    `json:"age"`
}

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// GetPetByID retrieves a pet by its ID
func (h *Handler) GetPetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		http.Error(w, "invalid pet id", http.StatusBadRequest)
		return
	}

	var pet Pet
	if result := h.db.First(&pet, id); result.Error != nil {
		http.Error(w, "pet not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pet)
}

// GetAllPets retrieves all pets
func (h *Handler) GetAllPets(w http.ResponseWriter, r *http.Request) {
	var pets []Pet
	if result := h.db.Find(&pets); result.Error != nil {
		http.Error(w, "failed to fetch pets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pets)
}

// CreatePet creates a new pet
func (h *Handler) CreatePet(w http.ResponseWriter, r *http.Request) {
	var pet Pet
	if err := json.NewDecoder(r.Body).Decode(&pet); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if result := h.db.Create(&pet); result.Error != nil {
		http.Error(w, "failed to create pet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pet)
}

// SetupRoutes sets up the pet routes on the provided router
func SetupRoutes(r *mux.Router, db *gorm.DB) {
	handler := NewHandler(db)

	petsRouter := r.PathPrefix("/pets").Subrouter()
	petsRouter.HandleFunc("", handler.GetAllPets).Methods("GET")
	petsRouter.HandleFunc("", handler.CreatePet).Methods("POST")
	petsRouter.HandleFunc("/{id}", handler.GetPetByID).Methods("GET")
}
