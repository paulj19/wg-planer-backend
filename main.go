package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Floor struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name"`
	Residents []string           `bson:"residents"`
	Tasks     []Task             `bson:"tasks"`
}

type Task struct {
	Name       string `bson:"name"`
	AssignedTo string `bson:"assignedTo"`
}

// type Resident struct {
//   Name       string `bson:"name"`
//   AssignedTo string `bson:"assignedTo"`
// }

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	http.HandleFunc("/floor", curdFloor)
	// id, err := insertNewFloor()
	// if err != nil {
	// 	log.Println("error inserting new floor", err)
	// }
	// fmt.Println(id)
	defer disconnectMongo(ctx)
	log.Println("Server running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func curdFloor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var floor Floor
		err := json.NewDecoder(r.Body).Decode(&floor)
		if err != nil {
			http.Error(w, "Error reading request body, bad format", http.StatusBadRequest)
			return
		}
		id, err := insertNewFloor(floor)
		if err != nil {
			http.Error(w, "Error inserting new floor", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": id})
		// json.NewEncoder(w).Encode(fmt.Sprintf("{\"id\": \"%s\"}", id))
	case http.MethodGet:
		floorId := r.URL.Path[len("/floor/"):]
		floor, err := getFloor(floorId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, "Floor not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Error getting floor", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(floor)
	}
}
