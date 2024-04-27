package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.Println("setting up test environment")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	os.Exit(m.Run())
}

func Test_InsertNewFloor(t *testing.T) {
	floor := `{
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
  ]}`

	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(addNewFloor)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	// body, err := strconv.Unquote(rr.Body.String())
	// if err != nil {
	// 	t.Errorf("error unquoting response body %s", err)
	// }
	// fmt.Println("response", body)

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
}

func Test_Return400WhenBadJsonFormat(t *testing.T) {
	floor := `{
	"name": "Floor1A",
	"residents": [
		"762b569bffebb4b815cd5e78",
		"762b5ace2337d3c989bcc238",
		"762b5f46cd8a580b287a8d84"
	],
	"tasks": [
			: "Gelbersack entfernen",
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
	]`

	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(addNewFloor)
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
