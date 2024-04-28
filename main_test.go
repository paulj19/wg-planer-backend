package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var floor = `{
  "name": "Floor1A",
  "residents": [
    "762b569bffebb4b815cd5e78",
    "762b5ace2337d3c989bcc238",
    "762b5f46cd8a580b287a8d84"
  ],
  "tasks": [
    {
      "name": "Gelbersack entfernen",
      "assignedTo": "662b5f46cd8a580b287a8d84"
    },
    {
      "name": "Biomüll wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    },
    {
      "name": "Restmull wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    }
  ],
  "rooms": [
    {
      "number": 1,
			"order": 1,
      "resident": "762b569bffebb4b815cd5e78"
    },
    {
      "number": 2,
			"order": 2,
      "resident": "762b5ace2337d3c989bcc238"
    },
    {
      "number": 3,
			"order": 3,
      "resident": "762b5f46cd8a580b287a8d84"
    }
  ]
}`

func TestMain(m *testing.M) {
	log.Println("setting up test environment")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	os.Exit(m.Run())
}

func Test_InsertNewFloor(t *testing.T) {
	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(curdFloor)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]string
	err = json.Unmarshal([]byte(rr.Body.String()), &response)
	if err != nil {
		fmt.Printf("handler returned invalid json %s", err.Error())
		t.Errorf("handler returned invalid json")
	}
	if _, ok := response["id"]; !ok {
		t.Errorf("handler returned invalid json")
	}
	if len(response["id"]) != 24 {
		t.Errorf("handler returned invalid id")
	}
	var floor_ Floor
	json.Unmarshal([]byte(floor), &floor_)
	insertedFloor, err := getFloor(response["id"])
	responseId, err := primitive.ObjectIDFromHex(response["id"])
	if err != nil {
		t.Fatal(err)
	}
	floor_.Id = responseId

	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(insertedFloor, floor_) {
		t.Errorf("handler returned wrong body: got %v want %v", insertedFloor, floor_)
	}
}

func Test_Return400WhenBadJsonFormat(t *testing.T) {
	floor_ := floor[:len(floor)-1]
	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor_))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(curdFloor)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
	if rr.Body.String() != "Error reading request body, bad format\n" {
		t.Errorf("handler returned wrong message: got %v want %v",
			rr.Body.String(), "Error reading request body, bad format\n")
	}
}

func Test_GetExistingFloor(t *testing.T) {
	var floor_ Floor
	json.Unmarshal([]byte(floor), &floor_)
	floorId, err := insertNewFloor(floor_)

	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "/floor/"+floorId, nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(curdFloor)
	handler.ServeHTTP(rr, req)
	fmt.Println(rr.Body.String())

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response Floor
	err = json.Unmarshal([]byte(rr.Body.String()), &response)
	if err != nil {
		fmt.Printf("handler returned invalid json %s", err.Error())
		t.Errorf("handler returned invalid json")
	}

	responseId, err := primitive.ObjectIDFromHex(floorId)
	if err != nil {
		t.Fatal(err)
	}
	floor_.Id = responseId

	if !reflect.DeepEqual(response, floor_) {
		t.Errorf("handler returned wrong body: got %v want %v", response, floor_)
	}
}
