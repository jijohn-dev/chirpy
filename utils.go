package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jijohn-dev/chirpy/internal/database"
)

type User struct {
	Id            uuid.UUID `json:"id"`
	Created_at    time.Time `json:"created_at"`
	Updated_at    time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	Is_chirpy_red bool      `json:"is_chirpy_red"`
}

type Chirp struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	UserId     uuid.UUID `json:"user_id"`
}

type userParams struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

func bindChirp(c database.Chirp) Chirp {
	chirp := Chirp{
		Id:         c.ID,
		Created_at: c.CreatedAt,
		Updated_at: c.UpdatedAt,
		Body:       c.Body,
		UserId:     c.UserID,
	}
	return chirp
}

func bindUser(u database.User) User {
	user := User{
		Id:            u.ID,
		Created_at:    u.CreatedAt,
		Updated_at:    u.UpdatedAt,
		Email:         u.Email,
		Is_chirpy_red: u.IsChirpyRed.Bool,
	}
	return user
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorVals struct {
		Error string `json:"error"`
	}
	respBody := errorVals{
		Error: msg,
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}

func cleanPost(body string) string {
	includedWords := []string{}
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(body, " ")

	for _, word := range words {
		if slices.Contains(badWords, strings.ToLower(word)) {
			includedWords = append(includedWords, "****")
		} else {
			includedWords = append(includedWords, word)
		}
	}
	return strings.Join(includedWords, " ")
}
