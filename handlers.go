package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AliBa1/chirpy/internal/auth"
	"github.com/AliBa1/chirpy/internal/database"
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

// func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
// 	refreshToken, err := auth.GetBearerToken(r.Header)
// 	if err != nil {
// 		log.Printf("error getting refresh token: %s", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}
// 	dbRefreshToken, err := cfg.dbQueries.GetRefreshToken(r.Context(), refreshToken)
// 	if err != nil {
// 		log.Printf("error getting refresh token from db: %s", err)
// 		w.WriteHeader(http.StatusUnauthorized)
// 		return
// 	}
//
// 	type response struct {
// 		Token string `json:"token"`
// 	}
// 	res := response{
// 		Token: dbRefreshToken.Token,
// 	}
// 	resJSON, err := json.Marshal(res)
// 	if err != nil {
// 		log.Printf("error getting token into JSON: %s", err)
// 		w.WriteHeader(http.StatusUnauthorized)
// 		return
// 	}
//
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write(resJSON)
// }

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
