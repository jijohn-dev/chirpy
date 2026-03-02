package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jijohn-dev/chirpy/internal/auth"
	"github.com/jijohn-dev/chirpy/internal/database"
)

func (cfg *apiConfig) handlerUsersCreate(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := userParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)

	createParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}

	user, err := cfg.db.CreateUser(r.Context(), createParams)

	if err != nil {
		log.Fatalf("Error creating user: %s", err)
	}

	res := bindUser(user)

	respondWithJSON(w, 201, res)
}

func (cfg *apiConfig) handlerUsersUpdate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email       string `json:"email"`
		NewPassword string `json:"password"`
	}

	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Failed to get token from header")
		log.Printf("error getting bearer token: %s", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		log.Printf("failed to validate JWT: %s", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		log.Printf("error hashing password: %s", err)
		return
	}

	updateParams := database.UpdateUserParams{
		HashedPassword: hashedPassword,
		Email:          params.Email,
		ID:             userID,
	}

	user, err := cfg.db.UpdateUser(r.Context(), updateParams)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 500, "Server error")
			log.Printf("User not found: %s", userID)
		}
		respondWithError(w, 500, "Server error")
		log.Printf("Error updating password: %s", err)
		return
	}

	res := bindUser(user)

	respondWithJSON(w, 200, res)
}
