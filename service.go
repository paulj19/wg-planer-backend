package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt"
)

type AuthService interface {
	getUserProfile(authToken string) (UserProfile, error)
	verifyToken(authToken string) (string, error)
}

type AuthServiceImpl struct {
	pubKey *rsa.PublicKey
}

func (as AuthServiceImpl) getUserProfile(authToken string) (UserProfile, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", "http://192.168.0.108:8082/userprofile", nil)
	req.Header.Add("Authorization", "Bearer "+authToken)
	if err != nil {
		return UserProfile{}, fmt.Errorf("Error creating http request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return UserProfile{}, fmt.Errorf("Error getting user profile: %w", err)
	}
	//convert response body to byte stream and then to string

	// Read the entire body into a byte slice
	// bodyBytes, err := io.ReadAll(resp.Body)
	// if err != nil {
	//   panic(err)
	// }

	// // Convert the byte slice to a string
	// bodyString := string(bodyBytes)

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return UserProfile{}, fmt.Errorf("Error getting user profile: %w", err)
	}

	var userProfile UserProfile
	err = json.NewDecoder(resp.Body).Decode(&userProfile)
	if err != nil {
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
	return claims["floor_id"].(string), nil
}
