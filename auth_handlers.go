package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/AliBa1/chirpy/internal/auth"
	"github.com/AliBa1/chirpy/internal/database"
	_ "github.com/lib/pq"
)

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		// ExpiresInSeconds *int   `json:"expires_in_seconds"`
	}

	var reqData request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqData)
	if err != nil {
		log.Printf("error decoding request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbUser, err := cfg.dbQueries.GetUserByEmail(r.Context(), reqData.Email)
	if err != nil {
		log.Printf("error getting user by email: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = auth.CheckPasswordHash(reqData.Password, dbUser.HashedPassword)
	if err != nil {
		log.Printf("password doesn't match hashed: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenExpiresIn := time.Hour
	// if reqData.ExpiresInSeconds != nil {
	// 	tokenExpiresIn = time.Duration(*reqData.ExpiresInSeconds)
	// }

	jwtToken, err := auth.MakeJWT(dbUser.ID, cfg.jwtSecret, tokenExpiresIn)
	if err != nil {
		log.Printf("error making JWT token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("error making refresh token: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	createRefTokenParams := database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: dbUser.ID,
	}
	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), createRefTokenParams)
	if err != nil {
		log.Printf("error creating refresh token in database: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		User         User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	res := response{
		User:         user,
		Token:        jwtToken,
		RefreshToken: refreshToken,
	}
	resJSON, err := json.Marshal(res)
	if err != nil {
		log.Printf("error marshaling user into JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resJSON)
}

func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("error getting refresh token: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.dbQueries.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("error getting user from refresh token: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		time.Hour,
	)
	if err != nil {
		log.Printf("error making JWT: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type response struct {
		Token string `json:"token"`
	}
	res := response{
		Token: accessToken,
	}
	resJSON, err := json.Marshal(res)
	if err != nil {
		log.Printf("error getting token into JSON: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resJSON)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("error getting refresh token: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = cfg.dbQueries.RevokeToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("error revoking refresh token: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
