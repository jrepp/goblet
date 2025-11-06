// Copyright 2025 Jacob Repp <jacobrepp@gmail.com>
//
// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main implements a CLI tool for getting tokens from Dex.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
)

var (
	dexURL       = flag.String("dex-url", "http://localhost:5556/dex", "Dex issuer URL")
	clientID     = flag.String("client-id", "goblet-cli", "OAuth2 client ID")
	clientSecret = flag.String("client-secret", "goblet-cli-secret", "OAuth2 client secret")
	redirectURL  = flag.String("redirect-url", "http://localhost:5555/callback", "OAuth2 redirect URL")
	outputFile   = flag.String("output", "./tokens/token.json", "Output file for token")
	listen       = flag.String("listen", ":5555", "Address to listen for OAuth2 callback")
)

// TokenResponse represents the token data.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	IDToken      string    `json:"id_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry"`
}

func main() {
	flag.Parse()

	ctx := context.Background()

	// Configure OAuth2
	config := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		RedirectURL:  *redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  *dexURL + "/auth",
			TokenURL: *dexURL + "/token",
		},
		Scopes: []string{"openid", "profile", "email", "groups"},
	}

	// Generate authorization URL
	state := "random-state-string"
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	// Start local server to receive callback
	tokenChan := make(chan *oauth2.Token)
	errChan := make(chan error)

	server := &http.Server{
		Addr:         *listen,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify state
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			errChan <- fmt.Errorf("invalid state")
			return
		}

		// Exchange code for token
		code := r.URL.Query().Get("code")
		token, err := config.Exchange(ctx, code)
		if err != nil {
			http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
			errChan <- fmt.Errorf("failed to exchange token: %w", err)
			return
		}

		_, _ = w.Write([]byte(`
<!DOCTYPE html>
<html>
<head><title>Goblet Authentication</title></head>
<body>
<h1>Authentication Successful!</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>
`))

		tokenChan <- token

		// Shutdown server after successful auth
		go func() {
			time.Sleep(1 * time.Second)
			_ = server.Shutdown(ctx)
		}()
	})

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Print instructions
	fmt.Println("Goblet Authentication")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Println("Please open the following URL in your browser:")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("Waiting for authentication...")

	// Wait for token or error
	select {
	case token := <-tokenChan:
		// Save token
		if err := saveToken(token); err != nil {
			log.Fatalf("Failed to save token: %v", err)
		}
		fmt.Println()
		fmt.Println("Authentication successful!")
		fmt.Printf("Token saved to: %s\n", *outputFile)
		fmt.Println()
		fmt.Println("To use this token with git:")
		fmt.Println("  export AUTH_TOKEN=$(jq -r .access_token " + *outputFile + ")")
		fmt.Println("  git -c \"http.extraHeader=Authorization: Bearer $AUTH_TOKEN\" fetch <url>")

	case err := <-errChan:
		log.Fatalf("Authentication failed: %v", err)

	case <-time.After(5 * time.Minute):
		log.Fatal("Authentication timed out")
	}
}

func saveToken(token *oauth2.Token) error {
	// Create directory if it doesn't exist
	outputDir := *outputFile
	if lastSlash := len(outputDir) - len("/token.json"); lastSlash > 0 {
		outputDir = outputDir[:lastSlash]
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Get ID token from extra data
	idToken, _ := token.Extra("id_token").(string)

	tokenResp := TokenResponse{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ExpiresIn:    int(time.Until(token.Expiry).Seconds()),
		IDToken:      idToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	data, err := json.MarshalIndent(tokenResp, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(*outputFile, data, 0600)
}
