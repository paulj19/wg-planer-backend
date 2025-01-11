package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
)

type AuthService interface {
	getUserProfile(r *http.Request) (UserProfile, error)
	verifyToken(authToken string) (string, error)
}

type AuthServiceImpl struct {
	pubKey *rsa.PublicKey
}

func (as AuthServiceImpl) getUserProfile(r *http.Request) (UserProfile, error) {
	authToken := r.Header.Get("Authorization")
	if authToken == "" {
		return UserProfile{}, fmt.Errorf("No auth token provided")
	}

	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", "http://172.17.0.5:8082/userprofile", nil)
	req.Header.Add("Authorization", authToken)
	if err != nil {
		logger.Error("Error creating http request", slog.Any("error", err))
		return UserProfile{}, fmt.Errorf("Error creating http request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		logger.Error("Error getting user profile", slog.Any("error", err))
		return UserProfile{}, fmt.Errorf("Error getting user profile: %w", err)
	}

	defer resp.Body.Close()
	var userProfile UserProfile
	err = json.NewDecoder(resp.Body).Decode(&userProfile)
	if err != nil {
		logger.Error("Error decoding user profile", slog.Any("error", err))
		return UserProfile{}, fmt.Errorf("Error decoding user profile: %w", err)
	}
	return userProfile, nil
}

func (as AuthServiceImpl) verifyToken(authToken string) (string, error) {
	var claims jwt.MapClaims
	token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
		return as.pubKey, nil
	})

	if err != nil {
		log.Println("Error parsing token:", err)
		return "", fmt.Errorf("Error parsing token: %w", err)
	}

	if token.Valid {
		claims = token.Claims.(jwt.MapClaims)
	} else {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				log.Println("Token is malformed")
			} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				log.Println("Token is expired or not active yet")
			} else {
				log.Println("Token is not valid:", err)
			}
		} else {
			log.Println("Token is not valid:", err)
		}
	}
	oid, ok := claims["oid"].(float64) // JWT claims are often parsed as float64
	if !ok {
		return "", fmt.Errorf("oid claim is not an int")
	}

	return strconv.Itoa(int(oid)), nil
}
