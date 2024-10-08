package huggingface

import (
	"errors"
	"os"
)

// Auth holds the authentication information
type Auth struct {
	token string
}

// NewAuthFromEnv creates a new Auth instance by reading the HF_TOKEN environment variable
func NewAuthFromEnv() (*Auth, error) {
	token := os.Getenv("HF_TOKEN")
	if token == "" {
		return nil, errors.New("environment variable HF_TOKEN is not set or empty")
	}
	return &Auth{token: token}, nil
}

// Header returns the authorization header
func (a *Auth) Header() string {
	return "Bearer " + a.token
}