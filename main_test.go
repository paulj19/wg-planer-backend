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

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var floor = `{
  "floorName": "Floor1A",
  "residents": [
    "762b569bffebb4b815cd5e78",
    "762b5ace2337d3c989bcc238",
    "762b5f46cd8a580b287a8d84"
  ],
  "tasks": [
    {
			"id": "1",
      "name": "Gelbersack entfernen",
      "assignedTo": "662b5f46cd8a580b287a8d84"
    },
    {
			"id": "2",
      "name": "Biomüll wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    },
    {
			"id": "3",
      "name": "Restmull wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    }
  ],
  "rooms": [
    {
			"id": "1",
      "number": "1",
			"order": 1,
      "resident": "762b569bffebb4b815cd5e78"
    },
    {
			"id": "2",
      "number": "2",
			"order": 2,
      "resident": "762b5ace2337d3c989bcc238"
    },
    {
			"id": "3",
      "number": "3",
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
	m := Maino{}
	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(m.curdFloor)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// if !reflect.DeepEqual(floor, rr.Body.String()) {
	// 	t.Errorf("handler returned wrong body: got %v want %v", rr.Body.String(), floor)
	// }
}

func Test_Return400WhenBadJsonFormat(t *testing.T) {
	m := Maino{}
	floor_ := floor[:len(floor)-1]
	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor_))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(m.curdFloor)
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
	floor, err := insertNewFloor(floor_)

	if err != nil {
		t.Fatal(err)
	}

	userprofile := UserProfile{
		Id:         "123",
		Username:   "paul",
		Email:      "paul@xxx.com",
		FloorId:    floor_.Id.String(),
		Oid:        "1",
		AuthServer: "HOME_BREW",
	}

	authServiceMock := new(AuthServiceMock)
	authServiceMock.On("getUserProfile", mock.Anything).Return(userprofile, nil)
	authServiceMock.On("verifyToken", mock.Anything).Return("", floor.Id.String(), nil)

	m := Maino{}
	m.initAuthService(authServiceMock)
	req, err := http.NewRequest("GET", "/floor", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(m.curdFloor)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response GetFloorResponse
	err = json.Unmarshal([]byte(rr.Body.String()), &response)
	if err != nil {
		t.Errorf("handler returned invalid json")
	}
	fmt.Println("RESPONSE", response)

	responseId, err := primitive.ObjectIDFromHex(floor.Id.Hex())
	if err != nil {
		t.Fatal(err)
	}
	floor_.Id = responseId
	getFloorResponse := GetFloorResponse{Floor: floor_, User: userprofile}

	if !reflect.DeepEqual(response, getFloorResponse) {
		t.Errorf("handler returned wrong body: got %v want %v", response, floor_)
	}
}

type AuthServiceMock struct {
	mock.Mock
}

func (as AuthServiceMock) getUserProfile(userId string) (UserProfile, error) {
	args := as.Called(userId)
	return args.Get(0).(UserProfile), args.Error(1)
}

func (as AuthServiceMock) verifyToken(r *http.Request) (string, string, error) {
	args := as.Called(r)
	return args.String(0), args.String(1), args.Error(2)
}
