package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Floor struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	FloorName string             `bson:"floorName,omitempty"`
	Residents []string           `bson:"residents,omitempty"`
	Tasks     []Task             `bson:"tasks,omitempty"`
	Rooms     []Room             `bson:"rooms,omitempty"`
}

type Task struct {
	Id         string `bson:"id,omitempty"`
	Name       string `bson:"name,omitempty"`
	AssignedTo string `bson:"assignedTo,omitempty"`
}

type Room struct {
	Id       string `bson:"id,omitempty"`
	Number   string `bson:"number,omitempty"`
	Order    int    `bson:"order,omitempty"`
	Resident string `bson:"resident,omitempty"`
}

type UserProfile struct {
	Id         string `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	FloorId    string `json:"floorId"`
	Oid        string `json:"oid"`
	AuthServer string `json:"authServer"`
}

type GetFloorResponse struct {
	Floor Floor       `json:"floor"`
	User  UserProfile `json:"user"`
}

// type Resident struct {
//   Name       string `bson:"name"`
//   AssignedTo string `bson:"assignedTo"`
// }

type Maino struct {
	authService AuthService
}

func main() {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	pubKey, err := initAuthServerPubKey()
	if err != nil {
		log.Fatal("Error initing public key", err)
	}
	m := Maino{}

	m.initAuthService(AuthServiceImpl{pubKey: pubKey})

	http.HandleFunc("/floor/", m.curdFloor)

	defer disconnectMongo(ctx)
	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (m Maino) initAuthService(as AuthService) {
	m.authService = as
}

func (m Maino) curdFloor(w http.ResponseWriter, r *http.Request) {
	corsHandler(w)
	switch r.Method {
	case http.MethodPost:
		var floor Floor
		err := json.NewDecoder(r.Body).Decode(&floor)
		if err != nil {
			http.Error(w, "Error reading request body, bad format", http.StatusBadRequest)
			return
		}
		newFloor, err := insertNewFloor(floor)
		if err != nil {
			http.Error(w, "Error inserting new floor", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newFloor)
	case http.MethodGet:
		userId, floorId, err := m.authService.verifyToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		floor, err := getFloor(floorId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, "Floor not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Error getting floor "+err.Error(), http.StatusInternalServerError)
			return
		}
		userprofile, err := m.authService.getUserProfile(userId)
		if err != nil {
			http.Error(w, "Error getting user profile "+err.Error(), http.StatusInternalServerError)
			return
		}
		if userprofile == (UserProfile{}) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		getFloorResponse := GetFloorResponse{Floor: floor, User: userprofile}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(getFloorResponse)
	case http.MethodOptions:
		fmt.Println("OPTIONS FLOOR", r.URL.Path)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
	}
}

func initAuthServerPubKey() (*rsa.PublicKey, error) {
	jwksJSON := `{"keys":[{"kty":"RSA","e":"AQAB","kid":"88d798cc-358f-491d-9fbd-8e52e165fc59","n":"pMusbA_4YWpIV9jRvghXxCK11gLJM90kRGMd6wRatT7MRZHlGdt9nPjN_kLn051Dy5cH7wbqYzZYvWxRjvHPG_dWJ115G6ddX16BMe8sb-HUMWvx39sA3t5I9GtaZjbkQy5NH0W7147s4NF_96eUQ2qzaAKpgA3GcHQ1iLqtr4VZgIn5R9RWO_8Uc81MuLIs08_sBnP84rECSB0LN3hi9_KMHX-PyshvGiyB6RrvHuq3QIQZRnrvDhFRjLlounJ5CErHC8aDpcxkjnj70wSuZnSsD73V2Yo4_-5Ou6CtYHCSCULE1uSojhNMBNhQL3OE-N6YeaKrlXY_JwhLaCBHPQ"}]}`

	// Parse the JWKS
	var jwks map[string][]map[string]interface{}
	if err := json.Unmarshal([]byte(jwksJSON), &jwks); err != nil {
		fmt.Println("Error parsing JWKS:", err)
		return nil, fmt.Errorf("Error initing pub key, parsing failed: %w", err)
	}

	// Extract the public key
	var pubKey *rsa.PublicKey
	for _, key := range jwks["keys"] {
		if key["kid"].(string) == "88d798cc-358f-491d-9fbd-8e52e165fc59" {
			modulus := key["n"].(string)
			exponent := key["e"].(string)
			n, err := base64.RawURLEncoding.DecodeString(modulus)
			if err != nil {
				log.Println("Error decoding modulus:", err)
				return nil, fmt.Errorf("Error initing pub key, extract public key failed: %w", err)
			}
			e, err := base64.RawURLEncoding.DecodeString(exponent)
			if err != nil {
				fmt.Println("Error decoding exponent:", err)
				return nil, fmt.Errorf("Error initing pub key, error decoding exponent: %w", err)
			}
			pubKey = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
			break
		}
	}

	if pubKey == nil {
		fmt.Println("Public key not found")
		return nil, fmt.Errorf("public key not found")
	}
	return pubKey, nil
}

func corsHandler(w http.ResponseWriter) {
	headers := w.Header()
	headers.Add("Access-Control-Allow-Origin", "*")
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")
	headers.Add("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, token")
	headers.Add("Access-Control-Allow-Methods", "GET, POST,OPTIONS")
}

func loadPublicKey(pemEncodedKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemEncodedKey))
	if block == nil {
		return nil, errors.New("failed to decode PEM-encoded public key")
	}
	key, _ := x509.ParsePKCS1PublicKey(block.Bytes)
	return key, nil
}
