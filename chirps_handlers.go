package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AliBa1/chirpy/internal/auth"
	"github.com/AliBa1/chirpy/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func (cfg *apiConfig) postChirpHandler(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("error getting bearer token: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
	if err != nil {
		log.Printf("invalid JWT token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	type requestChirp struct {
		Body string `json:"body"`
		// UserID uuid.UUID `json:"user_id"`
	}

	var reqChirp requestChirp
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&reqChirp)
	if err != nil {
		log.Printf("error decoding request for new chirp: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newChirpParams := database.CreateChirpParams{
		Body:   reqChirp.Body,
		UserID: userID,
	}
	dbChirp, err := cfg.dbQueries.CreateChirp(r.Context(), newChirpParams)
	if err != nil {
		log.Printf("error creating chirp in the database: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
	chirpJSON, err := json.Marshal(chirp)
	if err != nil {
		log.Printf("error marshaling chirp into JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(chirpJSON)
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		log.Printf("error getting chirps from db: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var chirps []Chirp
	for _, chirp := range dbChirps {
		cToAdd := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		chirps = append(chirps, cToAdd)
	}

	chirpsJSON, err := json.Marshal(chirps)
	if err != nil {
		log.Printf("error decoding chirps into JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(chirpsJSON)
}

func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("id")
	if chirpIDStr == "" {
		log.Print("chirp ID was empty or left out\n")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("Chirp ID:", chirpIDStr)

	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		log.Printf("error converting chirp ID string into UUID: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbChirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		log.Printf("chirp was not found: %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	chirpJSON, err := json.Marshal(chirp)
	if err != nil {
		log.Printf("error decoding chirp into JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(chirpJSON)
}
