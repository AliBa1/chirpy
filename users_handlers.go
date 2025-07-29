package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AliBa1/chirpy/internal/auth"
	"github.com/AliBa1/chirpy/internal/database"
	_ "github.com/lib/pq"
)

func (cfg *apiConfig) postUsersHandler(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var reqData request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqData)
	if err != nil {
		log.Printf("error decoding request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	reqData.Password, err = auth.HashPassword(reqData.Password)
	if err != nil {
		log.Printf("error hashing password: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	createUserParams := database.CreateUserParams{
		Email:          reqData.Email,
		HashedPassword: reqData.Password,
	}
	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), createUserParams)
	if err != nil {
		log.Printf("error creating user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := User{
		ID:          dbUser.ID,
		CreatedAt:   dbUser.CreatedAt,
		UpdatedAt:   dbUser.UpdatedAt,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed.Bool,
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

func (cfg *apiConfig) putUsersHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil || accessToken == "" {
		log.Printf("error getting access token: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		log.Printf("error getting user id from access token: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var reqData request
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&reqData)
	if err != nil {
		log.Printf("error decoding request data: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashedPassword, err := auth.HashPassword(reqData.Password)
	if err != nil {
		log.Printf("error hashing password: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	updateUserParams := database.UpdateUserParams{
		Email:          reqData.Email,
		HashedPassword: hashedPassword,
		ID:             userID,
	}
	dbUpdatedUser, err := cfg.dbQueries.UpdateUser(r.Context(), updateUserParams)
	if err != nil {
		log.Printf("error updating user: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	updatedUser := User{
		ID:          dbUpdatedUser.ID,
		CreatedAt:   dbUpdatedUser.CreatedAt,
		UpdatedAt:   dbUpdatedUser.UpdatedAt,
		Email:       dbUpdatedUser.Email,
		IsChirpyRed: dbUpdatedUser.IsChirpyRed.Bool,
	}
	updatedUserJSON, err := json.Marshal(updatedUser)
	if err != nil {
		log.Printf("error marshaling user into json: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(updatedUserJSON)
}
