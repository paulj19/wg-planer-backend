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

	// "github.com/golang-jwt/jwt"
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

// type Resident struct {
//   Name       string `bson:"name"`
//   AssignedTo string `bson:"assignedTo"`
// }

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	// rsaPublicKey, err := loadPublicKey(`{"keys":[{"kty":"RSA","e":"AQAB","kid":"822bdf10-226a-464e-bfe8-ee91020ab75a","n":"ukiXDTRqaUBBi7CeKMuxuybbPvgowrs2qQ-CJvdfGSmWka4YgENw9zoeHYhIPisOzpgKsWUQEpagEMhHibS1HkdBOeiAoNF4KgET4pKrw-FxKOGWk1YPkcySGlxuCRaV24h6Se8gqFr61523Qc_0g4mQc262dWghXRsmH71QZTg25YbQdTIaDjhu_5_MVK8vwtX-dL7SbCIevuRrSaZtw1_PLmMjL_q_rk8dyu4-7mYHCvhOD5D3NJ8PwY3fdu3XDNtx0t5KxqlJxdKgWVHbeiaWIHoMZXdI-AlqxAmbVQCo-yvwoZE_82DsVa5llYPb4A8Ap51vyvrwnaP5hu_9dQ"}]}`)
	// if err != nil {
	// 	log.Fatal("rsa key of authserver could not be parsed")
	// }
	jwksJSON := `{"keys":[{"kty":"RSA","e":"AQAB","kid":"88d798cc-358f-491d-9fbd-8e52e165fc59","n":"pMusbA_4YWpIV9jRvghXxCK11gLJM90kRGMd6wRatT7MRZHlGdt9nPjN_kLn051Dy5cH7wbqYzZYvWxRjvHPG_dWJ115G6ddX16BMe8sb-HUMWvx39sA3t5I9GtaZjbkQy5NH0W7147s4NF_96eUQ2qzaAKpgA3GcHQ1iLqtr4VZgIn5R9RWO_8Uc81MuLIs08_sBnP84rECSB0LN3hi9_KMHX-PyshvGiyB6RrvHuq3QIQZRnrvDhFRjLlounJ5CErHC8aDpcxkjnj70wSuZnSsD73V2Yo4_-5Ou6CtYHCSCULE1uSojhNMBNhQL3OE-N6YeaKrlXY_JwhLaCBHPQ"}]}`

	// Parse the JWKS
	var jwks map[string][]map[string]interface{}
	if err := json.Unmarshal([]byte(jwksJSON), &jwks); err != nil {
		fmt.Println("Error parsing JWKS:", err)
		return
	}

	// Extract the public key
	var pubKey *rsa.PublicKey
	for _, key := range jwks["keys"] {
		if key["kid"].(string) == "88d798cc-358f-491d-9fbd-8e52e165fc59" {
			modulus := key["n"].(string)
			exponent := key["e"].(string)
			n, err := base64.RawURLEncoding.DecodeString(modulus)
			if err != nil {
				fmt.Println("Error decoding modulus:", err)
				return
			}
			e, err := base64.RawURLEncoding.DecodeString(exponent)
			if err != nil {
				fmt.Println("Error decoding exponent:", err)
				return
			}
			pubKey = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
			break
		}
	}

	if pubKey == nil {
		fmt.Println("Public key not found")
		return
	}
	// mux := http.NewServeMux()
	// mux.HandleFunc("/getfloor/", getFloor_)
	// mux.HandleFunc("/floor", postFloor)
	fmt.Println("RSA PUBLIC KEY", pubKey)
	// Verify JWT token
	// tokenString := `eyJraWQiOiI4OGQ3OThjYy0zNThmLTQ5MWQtOWZiZC04ZTUyZTE2NWZjNTkiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJQYXVsb28iLCJhdWQiOiJ3Zy1wbGFuZXIiLCJuYmYiOjE3MTU3Njk3ODcsInNjb3BlIjpbIm9wZW5pZCJdLCJpc3MiOiJodHRwOi8vMTkyLjE2OC4zMy4xMzM6ODA4MSIsImZsb29yX2lkIjoiNjY0NDg2YmYwZDdlYTg4ZmU3NjYxYzBkIiwib2lkIjoxLCJleHAiOjE3MTU3NzE1ODcsImlhdCI6MTcxNTc2OTc4N30.KZDg0JZXC1iz7cDzP4VLDZUuXb7G-6tQzB3sJ-ruMJTGVcgbO6Z9Y45VXKUy7BzVkyy6UG11cF2ME8C-nFNi5QkbTOTu71GWLCra4KR6dlSN4mjjTqBAVf1O6ht9BRQ9aYNzKtvMFoUgOYkPwGDgE9oSu69M-TJe_z_dfkxn5QrEo1LGDE55JaUPwnukLsAsy97GgyFspYbagl9O_TG4uxKciG85Py62O_cNdjKAgJcDDyhmKJhXNO3Nba4VYZ83z5o-hE4mnDzSt__-RbK7x3lit6oZPGsERKXWoqTNp0SLPYP2TDIhpREkrQEEtzknSj2xEk6cWKG7JFngxZe5fw`
	// token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
	// 	return pubKey, nil
	// })

	// fmt.Println("token OOO", token, "WTF", err, "VAlid", token.Valid)
	// if err != nil {
	// 	fmt.Println("Error parsing token:", err)
	// 	return
	// }

	// if token.Valid {
	// 	claims := token.Claims.(jwt.MapClaims)
	// 	fmt.Println("JWT Token is valid")
	// 	fmt.Println("Claims:", claims)
	// } else if ve, ok := err.(*jwt.ValidationError); ok {
	// 	if ve.Errors&jwt.ValidationErrorMalformed != 0 {
	// 		fmt.Println("Token is malformed")
	// 	} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
	// 		fmt.Println("Token is expired or not active yet")
	// 	} else {
	// 		fmt.Println("Token is not valid:", err)
	// 	}
	// } else {
	// 	fmt.Println("Token is not valid:", err)
	// }
	http.HandleFunc("/floor/", postFloor)
	//allow CORS
	// id, err := insertNewFloor()
	// if err != nil {
	// 	log.Println("error inserting new floor", err)
	// }
	// fmt.Println(id)
	defer disconnectMongo(ctx)
	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getFloor_(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET FLOOR", r.URL.Path)
	floorId := r.URL.Path[len("/getfloor/"):]
	floor, err := getFloor(floorId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Floor not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error getting floor "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(floor)
}

func postFloor(w http.ResponseWriter, r *http.Request) {
	fmt.Println("CURD FLOOR", r.Method, r.URL.Path, r.Body)
	log.Println("CURD FLOOR", r.Method, r.URL.Path, r.Body)
	corsHandler(w)
	switch r.Method {
	case http.MethodPost:
		fmt.Println("POST FLOOR", r.URL.Path)
		var floor Floor
		// yyy := "{\"floorName\":\"\",\"tasks\":[{\"id\":0,\"name\":\"I\"}],\"rooms\":[{\"id\":0,\"order\":0,\"number\":\"R\"},{\"id\":1,\"order\":1,\"number\":\"W\"}]}"
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
		floorId := r.URL.Path[len("/floor/"):]
		fmt.Println("GET FLOOR", floorId, r.URL.Path)
		floor, err := getFloor(floorId)
		fmt.Println("GET FLOOR", floor)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, "Floor not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Error getting floor "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(floor)
	case http.MethodOptions:
		fmt.Println("OPTIONS FLOOR", r.URL.Path)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
	}
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

// func validateToken(tokenString string, publicKey *rsa.PublicKey) (bool, error) {
// 	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
// 		return publicKey, nil
// 	})
// 	if err != nil {
// 		return false, err
// 	}
// 	return token.Valid, nil
// }

// func validateTokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		authHeader := r.Header.Get("Authorization")
// 		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
// 			http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
// 			return
// 		}
// 		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
// 		valid, err := validateToken(tokenString, tokenString)
// 		if err != nil {
// 			http.Error(w, "Invalid token", http.StatusUnauthorized)
// 			return
// 		}
// 		if !valid {
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 			return
// 		}
// 		// Continue processing the request
// 		next.ServeHTTP(w, r)
// 	}
// }
