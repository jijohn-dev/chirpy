package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHash(t *testing.T) {
	password := "openSesame"
	hash, err := HashPassword(password)
	if err != nil {
		t.Errorf("Error hashing password: %s", err)
	}
	fmt.Println(hash)
}

func TestCompare(t *testing.T) {
	password := "openSesame"
	hash, err := HashPassword(password)
	if err != nil {
		t.Errorf("Error hashing password: %s", err)
	}

	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Errorf("Error comparing hash: %s", err)
	}

	if !match {
		t.Errorf("Match: false, expected: true")
	}
}

func TestMakeJWT(t *testing.T) {
	fmt.Println("---Testing MakeJWT---")
	id, err := uuid.Parse("87c36a7c-8f03-4dbc-a555-dcd4b3b55893")
	if err != nil {
		t.Errorf("Error parsing id: %s", err)
	}
	secret := "secret-key"
	expire := time.Minute

	token, err := MakeJWT(id, secret, expire)
	if err != nil {
		t.Errorf("Error creating token: %s", err)
	}
	fmt.Println(token)

	parsedId, err := ValidateJWT(token, secret)
	if parsedId != id {
		t.Errorf("IDs do not match")
	}
	fmt.Printf("ID: %s\n", parsedId)
}

func TestExpiredToken(t *testing.T) {
	fmt.Println("---Testing expired token validation---")
	id, err := uuid.Parse("87c36a7c-8f03-4dbc-a555-dcd4b3b55893")
	if err != nil {
		t.Errorf("Error parsing id: %s", err)
	}
	secret := "secret-key"
	expire := time.Second

	token, err := MakeJWT(id, secret, expire)
	if err != nil {
		t.Errorf("Error creating token: %s", err)
	}
	fmt.Println(token)

	time.Sleep(2 * time.Second)

	parsedId, err := ValidateJWT(token, secret)
	if err == nil {
		t.Errorf("No error returned when validating expired token")
	}
	fmt.Printf("ID: %s, error: %s\n", parsedId, err)
}

func TestInvalidSecret(t *testing.T) {
	fmt.Println("---Testing invalid secret validation---")
	id, err := uuid.Parse("87c36a7c-8f03-4dbc-a555-dcd4b3b55893")
	if err != nil {
		t.Errorf("Error parsing id: %s", err)
	}
	secret := "secret-key"
	expire := time.Second

	token, err := MakeJWT(id, secret, expire)
	if err != nil {
		t.Errorf("Error creating token: %s", err)
	}
	fmt.Println(token)

	parsedId, err := ValidateJWT(token, "secret-key1")
	if err == nil {
		t.Errorf("No error returned when validating with invalid key")
	}
	fmt.Printf("ID: %s, error: %s\n", parsedId, err)
}

func TestGetBearerToken(t *testing.T) {
	fmt.Println("---Testing GetBearerToken---")
	headers := http.Header{
		"Authorization": []string{"Bearer abc123"},
	}
	tok, err := GetBearerToken(headers)
	if err != nil {
		t.Errorf("Error getting bearer token: %s", err)
	}
	if tok != "abc123" {
		t.Errorf("Got: %s, expected: abc123", tok)
	}
	fmt.Println(tok)
}

func TestGetBearerTokenInvalidHeader(t *testing.T) {
	fmt.Println("---Testing GetBearerToken with invalid header---")
	headers := http.Header{}
	tok, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf("No error returned with empty header")
	}
	fmt.Println(tok)
}
