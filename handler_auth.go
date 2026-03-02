package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jijohn-dev/chirpy/internal/auth"
	"github.com/jijohn-dev/chirpy/internal/database"
)

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := cfg.db.GetUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	expirationTime := time.Hour

	token, err := auth.MakeJWT(user.ID, cfg.secret, expirationTime)
	if err != nil {
		respondWithError(w, 500, "Failed to create access JWT")
		return
	}

	// insert refresh token into DB
	refreshToken := auth.MakeRefreshToken()
	refreshParams := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	}
	refreshTokenEntry, err := cfg.db.CreateRefreshToken(r.Context(), refreshParams)

	res := response{
		User:         bindUser(user),
		Token:        token,
		RefreshToken: refreshTokenEntry.Token,
	}

	respondWithJSON(w, 200, res)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	// get refresh token from header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 500, "Failed to get token from header")
		log.Printf("error getting bearer token: %s", err)
		return
	}

	// look up refresh token in database
	refreshTokenEntry, err := cfg.db.GetRefreshTokenFromToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 401, "Refresh token not found")
			return
		}
		respondWithError(w, 500, "Error looking up refresh token")
		return
	}

	// check if token is revoked or expired
	if refreshTokenEntry.RevokedAt.Valid {
		respondWithError(w, 401, "Refresh token revoked")
		return
	}

	if time.Now().After(refreshTokenEntry.ExpiresAt) {
		respondWithError(w, 401, "Refresh token expired")
		return
	}

	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshTokenEntry.Token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 500, "Server error")
			log.Printf("User ID from refresh token not found")
		}
		respondWithError(w, 500, "Server error")
		log.Printf("Error looking up user from refresh token: %s", err)
		return
	}

	// create new access token
	accessToken, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Server error")
		log.Printf("error creating JWT: %s", err)
		return
	}

	res := response{
		accessToken,
	}

	respondWithJSON(w, 200, res)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	// get refresh token from header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 500, "Failed to get token from header")
		log.Printf("error getting bearer token: %s", err)
		return
	}

	_, err = cfg.db.RevokeRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, 500, "Server error")
		log.Printf("error revoking refresh token: %s", err)
		return
	}

	w.WriteHeader(204)
}
