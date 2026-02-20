package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jijohn-dev/chirpy/internal/auth"
	"github.com/jijohn-dev/chirpy/internal/database"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

type User struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Email      string    `json:"email"`
}

type Chirp struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	UserId     uuid.UUID `json:"user_id"`
}

type userParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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
		Id:         u.ID,
		Created_at: u.CreatedAt,
		Updated_at: u.UpdatedAt,
		Email:      u.Email,
	}
	return user
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := `<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
	</html>`
	w.Write([]byte(fmt.Sprintf(html, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}

	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
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

	res := User{
		Id:         user.ID,
		Created_at: user.CreatedAt,
		Updated_at: user.UpdatedAt,
		Email:      user.Email,
	}

	respondWithJSON(w, 201, res)
}

func (cfg *apiConfig) handlerChirpsCreate(w http.ResponseWriter, r *http.Request) {
	type postdata struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	post := postdata{}
	err := decoder.Decode(&post)
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
		UserID: post.UserID,
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

func (cfg *apiConfig) handlerChirpsGet(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps(r.Context())

	if err != nil {
		respondWithError(w, 500, "Error fetching chirps")
		log.Fatalf("Error fetching chirps: %s", err)
	}

	resChirps := []Chirp{}
	for _, c := range chirps {
		resC := bindChirp(c)
		resChirps = append(resChirps, resC)
	}

	respondWithJSON(w, 200, resChirps)
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

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	params := userParams{}
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

	res := bindUser(user)

	respondWithJSON(w, 200, res)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}

	dbQueries := database.New(db)

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", handlerHealthz)

	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	mux.HandleFunc("POST /api/users", apiCfg.handlerUsersCreate)

	mux.HandleFunc("POST /api/chirps", apiCfg.handlerChirpsCreate)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerChirpsGet)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerChirpGet)

	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
