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
	getUserProfile(userId string) (UserProfile, error)
	verifyToken(r *http.Request) (string, string, error)
}

type AuthServiceImpl struct {
	pubKey *rsa.PublicKey
}

func (as AuthServiceImpl) getUserProfile(userId string) (UserProfile, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8081/userprofile/"+userId, nil)
	if err != nil {
		return UserProfile{}, fmt.Errorf("Error creating http request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return UserProfile{}, fmt.Errorf("Error getting user profile: %w", err)
	}
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

func (as AuthServiceImpl) verifyToken(r *http.Request) (string, string, error) {
	authToken := r.Header.Get("Authorization")
	var claims jwt.MapClaims
	if authToken == "" {
		return "", "", fmt.Errorf("No token provided")
	}
	if authToken == "" {
		authToken = `eyJraWQiOiI4OGQ3OThjYy0zNThmLTQ5MWQtOWZiZC04ZTUyZTE2NWZjNTkiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJQYXVsb28iLCJhdWQiOiJ3Zy1wbGFuZXIiLCJuYmYiOjE3MTU3Njk3ODcsInNjb3BlIjpbIm9wZW5pZCJdLCJpc3MiOiJodHRwOi8vMTkyLjE2OC4zMy4xMzM6ODA4MSIsImZsb29yX2lkIjoiNjY0NDg2YmYwZDdlYTg4ZmU3NjYxYzBkIiwib2lkIjoxLCJleHAiOjE3MTU3NzE1ODcsImlhdCI6MTcxNTc2OTc4N30.KZDg0JZXC1iz7cDzP4VLDZUuXb7G-6tQzB3sJ-ruMJTGVcgbO6Z9Y45VXKUy7BzVkyy6UG11cF2ME8C-nFNi5QkbTOTu71GWLCra4KR6dlSN4mjjTqBAVf1O6ht9BRQ9aYNzKtvMFoUgOYkPwGDgE9oSu69M-TJe_z_dfkxn5QrEo1LGDE55JaUPwnukLsAsy97GgyFspYbagl9O_TG4uxKciG85Py62O_cNdjKAgJcDDyhmKJhXNO3Nba4VYZ83z5o-hE4mnDzSt__-RbK7x3lit6oZPGsERKXWoqTNp0SLPYP2TDIhpREkrQEEtzknSj2xEk6cWKG7JFngxZe5fw`
	}
	token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
		return as.pubKey, nil
	})

	if err != nil {
		fmt.Println("Error parsing token:", err)
		return "", "", fmt.Errorf("Error parsing token: %w", err)
	}

	if !token.Valid {
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
		claims = token.Claims.(jwt.MapClaims)
	}
	return claims["sub"].(string), claims["floorId"].(string), nil
}
