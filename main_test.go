package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"
)

var floorStub = `{
  "FloorName": "Awesome floor",
  "Tasks": [
    {
      "Id": "0",
      "Name": "Küche reinigen",
      "AssignedTo": null,
      "Reminders": 1,
      "AssignmentDate": "2024-06-13T14:48:00.000Z"
    },
    {
      "Id": "1",
      "Name": "Glastonne wegmachen",
      "AssignedTo": 1,
      "Reminders": 1,
      "AssignmentDate": "2024-05-16T14:48:00.000Z"
    },
    {
      "Id": "2",
      "Name": "Schwarz sack",
      "AssignedTo": 1,
      "Reminders": 0,
      "AssignmentDate": "2024-06-10T14:48:00.000Z"
    },
    {
      "Id": "3",
      "Name": "Mülxtonne wegbringen",
      "AssignedTo": 1,
      "Reminders": 2,
      "AssignmentDate": "2024-06-13T14:48:00.000Z"
    },
    {
      "Id": "4",
      "Name": "Gelbersack wegbringen",
      "AssignedTo": 1,
      "Reminders": 3,
      "AssignmentDate": "2024-06-19:48:00.000Z"
    },
    {
      "Id": "5",
      "Name": "Ofen Reinigen",
      "AssignedTo": 1,
      "Reminders": 4,
      "AssignmentDate": "2024-06-20T14:48:00.000Z"
    }
  ],
  "Rooms": [
    {
      "Id": 0,
      "Number": "301",
      "Order": 0,
      "Resident": {
        "Id": "1",
        "Name": "Max Musterman",
        "Available": true
      }
    },
    {
      "Id": 1,
      "Number": "302",
      "Order": 1,
      "Resident": {
        "Id": "2",
        "Name": "Leona Musterman",
        "Available": true
      }
    },
    {
      "Id": 2,
      "Number": "303",
      "Order": 2,
      "Resident": {
        "Id": "3",
        "Name": "Evelyn Weber",
        "Available": false
      }
    },
    {
      "Id": 3,
      "Number": "304",
      "Order": 3,
      "Resident": {
        "Id": "4",
        "Name": "Nodir Shirinov",
        "Available": true
      }
    },
    {
      "Id": 4,
      "Number": "305",
      "Order": 4,
      "Resident": {
        "Id": "5",
        "Name": "Benjamin Renert",
        "Available": false
      }
    },
    {
      "Id": 5,
      "Number": "306",
      "Order": 5,
      "Resident": {
        "Id": "6",
        "Name": "Abdul Majeed Nethyahu",
        "Available": true
      }
    },
    {
      "Id": 6,
      "Number": "307",
      "Order": 6,
      "Resident": null
    }
  ]
}
`

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

var FloorStub Floor

func TestMain(m *testing.M) {
	log.Println("setting up test environment")
	err := json.Unmarshal([]byte(floorStub), &f)
	if err != nil {
		log.Fatal("TestSetUp could not unmarshal FloorStub ", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	os.Exit(m.Run())
}

// func Test_InsertNewFloor(t *testing.T) {
// 	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(curdFloor)
// 	handler.ServeHTTP(rr, req)

// 	if status := rr.Code; status != http.StatusOK {
// 		t.Errorf("handler returned wrong status code: got %v want %v",
// 			status, http.StatusOK)
// 	}

// 	// if !reflect.DeepEqual(floor, rr.Body.String()) {
// 	// 	t.Errorf("handler returned wrong body: got %v want %v", rr.Body.String(), floor)
// 	// }
// }

// func Test_Return400WhenBadJsonFormat(t *testing.T) {
// 	floor_ := floor[:len(floor)-1]
// 	req, err := http.NewRequest("POST", "/floor", strings.NewReader(floor_))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(curdFloor)
// 	handler.ServeHTTP(rr, req)

// 	if status := rr.Code; status != http.StatusBadRequest {
// 		t.Errorf("handler returned wrong status code: got %v want %v",
// 			status, http.StatusBadRequest)
// 	}
// 	if rr.Body.String() != "Error reading request body, bad format\n" {
// 		t.Errorf("handler returned wrong message: got %v want %v",
// 			rr.Body.String(), "Error reading request body, bad format\n")
// 	}
// }

// func Test_GetExistingFloor(t *testing.T) {

// 	var floor_ Floor
// 	json.Unmarshal([]byte(floor), &floor_)
// 	floor, err := insertNewFloor(floor_)
// 	fmt.Println("floorId xXX", floor.Id)

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	userprofile := UserProfile{
// 		Id:         "123",
// 		Username:   "paul",
// 		Email:      "paul@xxx.com",
// 		FloorId:    floor_.Id.String(),
// 		Oid:        "1",
// 		AuthServer: "HOME_BREW",
// 	}

// 	authServiceMock := new(AuthServiceMock)
// 	authServiceMock.On("getUserProfile", mock.Anything).Return(userprofile, nil)
// 	authServiceMock.On("verifyToken", mock.Anything).Return("", floor.Id.String()[10:len(floor.Id.String())-2], nil)

// 	initAuthService(authServiceMock)
// 	req, err := http.NewRequest("GET", "/floor", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(curdFloor)
// 	handler.ServeHTTP(rr, req)

// 	if status := rr.Code; status != http.StatusOK {
// 		t.Errorf("handler returned wrong status code: got %v want %v",
// 			status, http.StatusOK)
// 	}

// 	var response GetFloorResponse
// 	err = json.Unmarshal([]byte(rr.Body.String()), &response)
// 	if err != nil {
// 		t.Errorf("handler returned invalid json")
// 	}
// 	fmt.Println("RESPONSE", response)

// 	responseId, err := primitive.ObjectIDFromHex(floor.Id.Hex())
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	floor_.Id = responseId
// 	getFloorResponse := GetFloorResponse{Floor: floor_, UserProfile: userprofile}

// 	if !reflect.DeepEqual(response, getFloorResponse) {
// 		t.Errorf("handler returned wrong body: got %v want %v", response, getFloorResponse)
// 	}
// }

// type AuthServiceMock struct {
// 	mock.Mock
// }

// func (as AuthServiceMock) getUserProfile(userId string) (UserProfile, error) {
// 	args := as.Called(userId)
// 	return args.Get(0).(UserProfile), args.Error(1)
// }

// func (as AuthServiceMock) verifyToken(r *http.Request) (string, string, error) {
// 	args := as.Called(r)
// 	return args.String(0), args.String(1), args.Error(2)
// }
