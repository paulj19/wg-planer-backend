package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// type Resident struct {
//   Name       string `bson:"name"`
//   AssignedTo string `bson:"assignedTo"`
// }

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	// mux := http.NewServeMux()
	// mux.HandleFunc("/getfloor/", getFloor_)
	// mux.HandleFunc("/floor", postFloor)
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
