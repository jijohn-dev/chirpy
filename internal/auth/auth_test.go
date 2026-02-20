package auth

import (
	"fmt"
	"testing"
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
