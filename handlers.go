package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/AliBa1/chirpy/internal/auth"
	"github.com/google/uuid"
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

func (cfg *apiConfig) polkaWebhookHandler(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil || apiKey != cfg.polkaKey {
		log.Printf("error getting api key or missing in header: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	type reqData struct {
		UserID string `json:"user_id"`
	}

	type requestStruct struct {
		Event string  `json:"event"`
		Data  reqData `json:"data"`
	}

	var request requestStruct
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&request)
	if err != nil {
		log.Printf("error decoding polka webhook request into json: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if request.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(request.Data.UserID)
	if err != nil {
		log.Printf("error parsing user id: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = cfg.dbQueries.UpgradeToChirpyRed(r.Context(), userID)
	if err != nil {
		log.Printf("error upgrading to chirpy red: %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
