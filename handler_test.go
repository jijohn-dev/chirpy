package main

import (
	"fmt"
	"testing"
)

func TestCleanPost(t *testing.T) {
	msg := "I really need a kerfuffle to go to bed sooner, Fornax !"
	cleaned := cleanPost(msg)
	fmt.Println(cleaned)

	expected := "I really need a **** to go to bed sooner, **** !"
	if cleaned != expected {
		t.Errorf("expected: %v, got: %v", expected, cleaned)
	}
}
