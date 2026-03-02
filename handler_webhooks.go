package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jijohn-dev/chirpy/internal/auth"
)

func (cfg *apiConfig) handlerPolka(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, 401, "invalid API key")
		log.Printf("error extracting API key: %s", err)
		return
	}

	if apiKey != cfg.polka_key {
		respondWithError(w, 401, "invalid API key")
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	if params.Event != "user.upgraded" {
		respondWithError(w, 204, "Unrecognized event")
		return
	}

	userID, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		log.Printf("error parsing user id: %s", err)
		return
	}

	_, err = cfg.db.UpgradeUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "User not found")
			log.Printf("User not found: %s", userID)
		}
		respondWithError(w, 500, "Server error")
		log.Printf("Error upgrading user: %s", err)
		return
	}

	w.WriteHeader(204)
}
