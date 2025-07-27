package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(
		`<html>
		  <body>
		    <h1>Welcome, Chirpy Admin</h1>
		    <p>Chirpy has been visited %d times!</p>
		  </body>
		</html>`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	err := cfg.dbQueries.ResetUsers(r.Context())
	if err != nil {
		log.Printf("error reseting users: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)

}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	requestData := request{}
	err := decoder.Decode(&requestData)
	if err != nil {
		log.Printf("Error decoding the request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		Cleaned string `json:"cleaned_body"`
		Error   string `json:"error"`
	}

	var responseData response
	var statusCode int
	if len(requestData.Body) > 140 {
		responseData.Error = "Chirp is too long"
		statusCode = http.StatusBadRequest
	} else {
		responseData.Cleaned = replaceProfanity(requestData.Body)
		statusCode = http.StatusOK
	}

	responseJSON, err := json.Marshal(responseData)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{'error': 'Something went wrong'}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(responseJSON)
}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email string `json:"email"`
	}

	var reqData request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqData)
	if err != nil {
		log.Printf("error decoding request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), reqData.Email)
	if err != nil {
		log.Printf("error creating user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		log.Printf("error marshaling user into JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(userJSON)
}
