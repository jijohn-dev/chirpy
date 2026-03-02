package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/jijohn-dev/chirpy/internal/auth"
	"github.com/jijohn-dev/chirpy/internal/database"
)

func (cfg *apiConfig) handlerChirpsCreate(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 500, "Failed to get token from header")
		log.Printf("error getting bearer token: %s", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		log.Printf("failed to validate JWT: %s", err)
		return
	}

	type postdata struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	post := postdata{}
	err = decoder.Decode(&post)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	if len(post.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	cleanedBody := cleanPost(post.Body)

	params := database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: userID,
	}

	chirp, err := cfg.db.CreateChirp(r.Context(), params)

	if err != nil {
		log.Fatalf("Error creating chirp: %s", err)
	}

	res := Chirp{
		Id:         chirp.ID,
		Created_at: chirp.CreatedAt,
		Updated_at: chirp.UpdatedAt,
		Body:       chirp.Body,
		UserId:     chirp.UserID,
	}

	respondWithJSON(w, 201, res)
}

func createChirpsResponse(chirps []database.Chirp, sortDirection string) []Chirp {
	resChirps := []Chirp{}
	for _, c := range chirps {
		resC := bindChirp(c)
		resChirps = append(resChirps, resC)
	}
	if sortDirection == "desc" {
		sort.Slice(resChirps, func(i, j int) bool { return resChirps[i].Created_at.After(resChirps[j].Created_at) })
	}
	return resChirps
}

func (cfg *apiConfig) handlerChirpsGet(w http.ResponseWriter, r *http.Request) {
	authorID := r.URL.Query().Get("author_id")
	sortDirection := r.URL.Query().Get("sort")

	if authorID == "" {
		chirps, err := cfg.db.GetChirps(r.Context())
		if err != nil {
			respondWithError(w, 500, "Error fetching chirps")
			log.Fatalf("Error fetching chirps: %s", err)
		}

		resChirps := createChirpsResponse(chirps, sortDirection)

		respondWithJSON(w, 200, resChirps)
	} else {
		userID, err := uuid.Parse(authorID)
		chirps, err := cfg.db.GetChirpsByAuthor(r.Context(), userID)
		if err != nil {
			respondWithError(w, 500, "Error fetching chirps")
			log.Fatalf("Error fetching chirps: %s", err)
		}

		resChirps := createChirpsResponse(chirps, sortDirection)

		respondWithJSON(w, 200, resChirps)
	}

}

func (cfg *apiConfig) handlerChirpGet(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))

	if err != nil {
		respondWithError(w, 400, "Error fetching chirp")
		log.Fatalf("Error parsing chirp ID (%s): %s", r.PathValue("chirpID"), err)
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "Chirp not found")
			return
		}
		respondWithError(w, 500, "Error fetching chirp")
		return
	}

	res := bindChirp(chirp)
	respondWithJSON(w, 200, res)
}

func (cfg *apiConfig) handlerChirpDelete(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))

	if err != nil {
		respondWithError(w, 400, "Error fetching chirp")
		log.Fatalf("Error parsing chirp ID (%s): %s", r.PathValue("chirpID"), err)
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

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "Chirp not found")
			return
		}
		respondWithError(w, 500, "Error fetching chirp")
		return
	}

	// check if user is author of chirp
	if userID != chirp.UserID {
		respondWithError(w, 403, "Unauthorized")
		return
	}

	chirp, err = cfg.db.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		log.Printf("error deleting chirp: %s", err)
		return
	}

	w.WriteHeader(204)
}
