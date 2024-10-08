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
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Floor struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	FloorName string             `bson:"floorName"`
	Residents []string           `bson:"residents"`
	Tasks     []Task             `bson:"tasks"`
	Rooms     []Room             `bson:"rooms"`
	Votings   []Voting           `bson:"votings"`
}

type Task struct {
	Id             string    `bson:"id"`
	Name           string    `bson:"name"`
	AssignedTo     int       `bson:"assignedTo"`
	Reminders      int       `bson:"reminders"`
	AssignmentDate time.Time `bson:"assignmentDate"`
}

type Room struct {
	Id       int      `bson:"id"`
	Number   string   `bson:"number"`
	Order    int      `bson:"order"`
	Resident Resident `bson:"resident"`
}

type Resident struct {
	Id            string `bson:"id"`
	Name          string `bson:"name"`
	Available     bool   `bson:"available"`
	ExpoPushToken string `bson:"expoPushToken"`
}

type Voting struct {
	Id           int           `bson:"id"`
	Type         string        `bson:"type"`
	Data         Task          `bson:"data"`
	Accepts      []string      `bson:"accepts"`
	Rejects      []string      `bson:"rejects"`
	LaunchDate   time.Time     `bson:"date"`
	VotingWindow time.Duration `bson:"votingWindow"`
	CreatedBy    string        `bson:"createdBy"`
}

type UserProfile struct {
	Id         int64  `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	FloorId    string `json:"floorId"`
	Oid        int64  `json:"oid"`
	AuthServer string `json:"authServer"`
}

type GetFloorResponse struct {
	Floor       Floor       `json:"floor"`
	UserProfile UserProfile `json:"userprofile"`
}

type RegisterTokenRequest struct {
	ExpoPushToken string `json:"expoPushToken"`
	FloorId       string `json:"floorId"`
	UserId        string `json:"userId"`
}

// type Resident struct {
//   Name       string `bson:"name"`
//   AssignedTo string `bson:"assignedTo"`
// }

var IsTest bool
var userId string
var floorId = "669fca69d244526d709f6d76"
var authService AuthService

type services struct {
	taskService TaskService
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func main() {
	//TODO handle panics so that the server does not shut down
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	services := services{taskService: TaskUpdateRequest{}}
	// pubKey, err := initAuthServerPubKey()
	// if err != nil {
	// 	log.Fatal("Error initing public key", err)
	// }

	// initAuthService(AuthServiceImpl{pubKey: pubKey})

	http.HandleFunc("/floor/", crudFloor)
	http.HandleFunc("/post-login", startupInfo)
	http.HandleFunc("/update-task", services.taskService.HandleTaskUpdate)
	http.HandleFunc("/register-expo-token", registerExpoPushToken)
	http.HandleFunc("/remind-task", services.taskService.HandleTaskRemind)
	http.HandleFunc("/update-availability", HandleAvailabilityStatusChange)
	http.HandleFunc("/generate-code", HandleCodeGeneration)
	http.HandleFunc("/submit-code", HandleCodeSubmit)
	http.HandleFunc("/add-newResident", HandleAddNewResident)
	http.HandleFunc("/create-del-task", HandleTaskCreateDelete)
	http.HandleFunc("/update-voting", HandleTaskVotingResponse)

	defer disconnectMongo(ctx)
	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initAuthService(as AuthService) {
	authService = as
}

func startupInfo(w http.ResponseWriter, r *http.Request) {
	corsHandler(w)
	// authToken := r.Header.Get("Authorization")
	// if authToken == "" {
	// 	http.Error(w, "No token provided", http.StatusUnauthorized)
	// }
	// authToken = authToken[7:]
	// floorId, err := authService.verifyToken(authToken)
	// if err != nil {
	// 	return
	// }

	floorId := "669fca69d244526d709f6d76"
	floor, err := FindFloor(floorId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Floor not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error getting floor "+err.Error(), http.StatusInternalServerError)
		return
	}

	userprofile := UserProfile{
		Id:         2,
		Username:   "Paulo",
		Email:      "maxmuster@gmail.com",
		FloorId:    "66603e2a00afb9bb44b3cadb",
		Oid:        1,
		AuthServer: "HOME_BREW",
	}
	// userprofile, err := authService.getUserProfile(authToken)
	// if err != nil {
	// 	http.Error(w, "Error getting user profile "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	if userprofile == (UserProfile{}) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	getFloorResponse := GetFloorResponse{Floor: floor, UserProfile: userprofile}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getFloorResponse)
}

func crudFloor(w http.ResponseWriter, r *http.Request) {
	corsHandler(w)
	switch r.Method {
	case http.MethodPost:
		var floor Floor
		err := json.NewDecoder(r.Body).Decode(&floor)
		if err != nil {
			fmt.Println("Error reading request body", err)
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
		floor, err := FindFloor(floorId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, "Floor not found", http.StatusNotFound)
				// http.Error(w, "Error getting floor "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(floor)
	case http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
	}
}

func registerExpoPushToken(w http.ResponseWriter, r *http.Request) {
	corsHandler(w)
	var registerTokenRequest RegisterTokenRequest
	err := json.NewDecoder(r.Body).Decode(&registerTokenRequest)
	if err != nil {
		logger.Error("registerTokenRequest", slog.Any("error", err))
		http.Error(w, fmt.Sprintf("Error reading request body %v", err), http.StatusBadRequest)
		return
	}
	floor, err := FindFloor(registerTokenRequest.FloorId)
	if err != nil {
		logger.Error("registerTokenRequest getFloor", slog.Any("error", err), slog.Any("registerTokenRequest", registerTokenRequest))
		http.Error(w, "Error getting floor", http.StatusUnprocessableEntity)
		return
	}
	var found bool
	var roomIndex int
	for i, room := range floor.Rooms {
		if room.Resident.Id == registerTokenRequest.UserId {
			floor.Rooms[i].Resident.ExpoPushToken = registerTokenRequest.ExpoPushToken
			roomIndex = i
			found = true
			break
		}

	}
	if !found {
		logger.Error("registerTokenRequest", slog.Any("error", "User not found in floor"), slog.Any("registerTokenRequest", registerTokenRequest), slog.Any("floor", floor))
		http.Error(w, "User not found in floor", http.StatusUnprocessableEntity)
		return
	}
	floor, err = updateExpoPushToken(floor, roomIndex)
	if err != nil {
		logger.Error("registerTokenRequest", slog.Any("error", err), slog.Any("registerTokenRequest", registerTokenRequest), slog.Any("floor", floor))
		http.Error(w, "Error updating expo push token in DB", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(floor)
}

func getJwksFromAuthServer() (map[string][]map[string]interface{}, error) {
	httpClient := &http.Client{}

	req, err := http.NewRequest("GET", "http://192.168.0.108:8081/oauth2/jwks", nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating http request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error getting JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error getting JWKS: %w", err)
	}
	// var jwks string
	var jwks map[string][]map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)
	if err != nil {
		return nil, fmt.Errorf("Error decoding JWKS: %w", err)
	}
	return jwks, nil
}

func initAuthServerPubKey() (*rsa.PublicKey, error) {
	// jwksJSON := `{"keys":[{"kty":"RSA","e":"AQAB","kid":"63c96dd9-bcca-45f8-8bad-744bb02f3872","n":"4IVGlvJZni-xZ7sgOetXegIKqA6ffQKAMOqp2TjO7b80o7oUGVmr7f6lwQ3L43HT9Lx-PRP5h61Zay3RaI47lsmCqBUHfuutp3ijVpeL5c1YDI9RUjEHrrgK78Rocx8LP2pXgl70TbL9275ugkcCSKm-9_qxTjTjO5azRqtQY0PCZmzt_kfmkNEEw7l6vjzPEY-CEk5EL-bp1g7UEkD3jdlif2fHGpb-Ql5KL7O3ytBt-c8LwDhhtCeFoyejK1p7L8BOr1xcaMVZuXNsDavbpPdh7ml6mSRxrBkSckY4Y2OB3SdOJMS_6CduZkz-LVi9RPian5xJVLmPcs2l_gU6mw"}]}`
	// jwksJSON := `{"keys":[{"kty":"RSA","e":"AQAB","kid":"4e807cc8-a4fe-4b2c-ba40-571f64c8517d","n":"wGJpVlli_km_JISEmamXdrDPASbZXys0yhCJCZncfmrTt9MM-tKQRJXpvSHK2rILVBtW4KOjguU42kfNHgNxS_xg6O5nsfa5jMsLOJg1lku8a56QA6xrLJ1_mNHFgX1B0psQTUkQXtVWZQZD1shnqNbOEDrwwxx1LbRWbb86KSZnVccPhSQOUxklP3HI64ZS0P3AQlAqDJ6bsRs3hqI12NcQalzALFHWCl0eqZZa19jL3XDqyfCzg8uJ3KJ5Vcvmj-b56aFised8WIhHBSO5ZsYYhjPABFcMaZIOdM5jM-QUGA1WfHV4mGmR6XDmfDsOnDru5xNFqlPMSSBGdTN9kw"}]}`
	jwks, err := getJwksFromAuthServer()
	if err != nil {
		return nil, fmt.Errorf("Error initing pub key, getting JWKS failed: %w", err)
	}

	// Parse the JWKS
	// var jwks map[string][]map[string]interface{}
	// if err := json.Unmarshal([]byte(jwksJSON), &jwks); err != nil {
	// 	log.Println("Error parsing JWKS:", err)
	// 	return nil, fmt.Errorf("Error initing pub key, parsing failed: %w", err)
	// }

	// Extract the public key
	var pubKey *rsa.PublicKey
	for _, key := range jwks["keys"] {
		modulus := key["n"].(string)
		exponent := key["e"].(string)
		n, err := base64.RawURLEncoding.DecodeString(modulus)
		if err != nil {
			log.Println("Error decoding modulus:", err)
			return nil, fmt.Errorf("Error initing pub key, extract public key failed: %w", err)
		}
		e, err := base64.RawURLEncoding.DecodeString(exponent)
		if err != nil {
			log.Println("Error decoding exponent:", err)
			return nil, fmt.Errorf("Error initing pub key, error decoding exponent: %w", err)
		}
		pubKey = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
		break
	}

	if pubKey == nil {
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
