package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var taskAssign = `{
	"floorId": "1",
	"task": {
		"id": "1",
		"assignedTo": "1"
	},
	"action": "ASSIGN",
	"nextRoom": 
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
}`

var floorsCreated []primitive.ObjectID

func Test_findTask(t *testing.T) {
	t.Run("should find task", func(t *testing.T) {
		task, _ := findTask(FloorStub.Tasks, FloorStub.Tasks[len(FloorStub.Tasks)-1].Id)
		if !reflect.DeepEqual(task, FloorStub.Tasks[len(FloorStub.Tasks)-1]) {
			t.Errorf("task not found: got %v want %v", task, FloorStub.Tasks[len(FloorStub.Tasks)-1])
		}
	})
	t.Run("should not find task", func(t *testing.T) {
		task, _ := findTask(FloorStub.Tasks, "9999")
		var taskNil Task
		if !reflect.DeepEqual(task, taskNil) {
			t.Error("findTask did not return zero value task")
		}
	})
}

func Test_nextAssignee(t *testing.T) {
	t.Run("should find next assignee", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    2,
					Order: 1,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    3,
					Order: 2,
					Resident: Resident{
						Available: true,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextRoom, err := nextAssignee(f, f.Tasks[0])
		if err != nil {
			t.Fatal(err)
		}
		if nextRoom.Id != 2 {
			t.Errorf("next assignee not found: got %v want %v", nextRoom.Id, f.Tasks[0].AssignedTo)
		}
	})
	t.Run("should not assign to non-avail resident", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    2,
					Order: 1,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    3,
					Order: 2,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    4,
					Order: 3,
					Resident: Resident{
						Available: true,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextRoom, err := nextAssignee(f, f.Tasks[0])
		if err != nil {
			t.Fatal(err)
		}
		if nextRoom.Id != 4 {
			t.Errorf("next assignee not found: got %v want %v", nextRoom.Id, f.Tasks[1].AssignedTo)
		}
	})
	t.Run("should not find next assignee", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
				{
					Id:    2,
					Order: 1,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    3,
					Order: 2,
					Resident: Resident{
						Available: false,
					},
				},
				{
					Id:    4,
					Order: 3,
					Resident: Resident{
						Available: false,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextAss, err := nextAssignee(f, f.Tasks[0])
		if err == nil || err.Error() != "No next assignee available" || !reflect.DeepEqual(nextAss, Room{}) {
			t.Errorf("no avail residents, nextAss should be emtpy room with an error")
		}
	})
	t.Run("should not find next assignee", func(t *testing.T) {
		f := Floor{
			Rooms: []Room{
				{
					Id:    1,
					Order: 0,
					Resident: Resident{
						Available: true,
					},
				},
			},
			Tasks: []Task{
				{
					Id:         "1",
					AssignedTo: 1,
				},
			},
		}
		nextAss, err := nextAssignee(f, f.Tasks[0])
		if err == nil || err.Error() != "No next assignee available" || !reflect.DeepEqual(nextAss, Room{}) {
			t.Errorf("no avail residents, nextAss should be emtpy room with an error")
		}
	})
}

func Test_updateTask(t *testing.T) {
	f, err := insertTestFloor(FloorStub)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("should assign task", func(t *testing.T) {
		fmt.Println("Floor XXX", f.Id)
		tuStub := TaskUpdate{
			FloorId:  f.Id.String()[10:34],
			Task:     FloorStub.Tasks[0],
			Action:   "ASSIGN",
			NextRoom: FloorStub.Rooms[3],
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != FloorStub.Rooms[3].Id {
			t.Fatalf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, FloorStub.Rooms[2].Id)
		}
	})

	t.Run("should 422 when resident unavailable after user click", func(t *testing.T) {
		FloorStub.Rooms[3].Resident.Available = false
		tuStub := TaskUpdate{
			FloorId:  f.Id.String()[10:34],
			Task:     FloorStub.Tasks[0],
			Action:   "ASSIGN",
			NextRoom: FloorStub.Rooms[3],
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnprocessableEntity)
		}
	})
	t.Run("should return 422 due to ass unavail", func(t *testing.T) {
		tuStub := TaskUpdate{
			FloorId:  f.Id.String()[10:34],
			Task:     FloorStub.Tasks[0],
			Action:   "ASSIGN",
			NextRoom: FloorStub.Rooms[2],
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnprocessableEntity)
		}
	})

	t.Run("should reassign done task", func(t *testing.T) {
		var floorStub = `{
  		"FloorName": "Awesome floor",
  		"Tasks": [
  		  {
  		    "Id": "0",
  		    "Name": "Küche reinigen",
  		    "AssignedTo": 0,
  		    "Reminders": 1,
  		    "AssignmentDate": "2024-06-13T14:48:00.000Z"
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
  		  }
  		]
}
`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Fatal("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("Floor YYY", FloorStub.Id)
		floorsCreated = append(floorsCreated, f.Id)
		tuStub := TaskUpdate{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)
		fmt.Println("updatedFloor", updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != f.Rooms[1].Id {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, f.Rooms[1].Id)
		}
	})
	t.Run("should not assign to non-avail resident when task done", func(t *testing.T) {
		var floorStub = `{
  		"FloorName": "Awesome floor",
  		"Tasks": [
  		  {
  		    "Id": "0",
  		    "Name": "Küche reinigen",
  		    "AssignedTo": 1,
  		    "Reminders": 1,
  		    "AssignmentDate": "2024-06-13T14:48:00.000Z"
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
  		      "Available": false
  		    }
  		  },
    		{
    		  "Id": 2,
    		  "Number": "303",
    		  "Order": 2,
    		  "Resident": {
    		    "Id": "3",
    		    "Name": "Evelyn Weber",
    		    "Available": true
    		  }
    		}
  		]
}
`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Fatal("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Fatal(err)
		}
		tuStub := TaskUpdate{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != f.Rooms[2].Id {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, f.Rooms[2].Id)
		}
	})

	t.Run("should not assign to any resident DONE", func(t *testing.T) {
		var floorStub = `{
  		"FloorName": "Awesome floor",
  		"Tasks": [
  		  {
  		    "Id": "0",
  		    "Name": "Küche reinigen",
  		    "AssignedTo": 0,
  		    "Reminders": 1,
  		    "AssignmentDate": "2024-06-13T14:48:00.000Z"
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
  		      "Available": false
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
    		}
  		]
}
`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Fatal("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Fatal(err)
		}
		tuStub := TaskUpdate{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != -1 {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, -1)
		}
	})

	t.Run("should not assign when task done and no other residents", func(t *testing.T) {
		var floorStub = `{
  		"FloorName": "Awesome floor",
  		"Tasks": [
  		  {
  		    "Id": "0",
  		    "Name": "Küche reinigen",
  		    "AssignedTo": 0,
  		    "Reminders": 1,
  		    "AssignmentDate": "2024-06-13T14:48:00.000Z"
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
  		  }
  		]
}
`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Fatal("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Fatal(err)
		}
		tuStub := TaskUpdate{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != -1 {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, -1)
		}
	})

	t.Run("should done with cycled to first", func(t *testing.T) {
		var floorStub = `{
  		"FloorName": "Awesome floor",
  		"Tasks": [
  		  {
  		    "Id": "0",
  		    "Name": "Küche reinigen",
  		    "AssignedTo": 2,
  		    "Reminders": 1,
  		    "AssignmentDate": "2024-06-13T14:48:00.000Z"
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
    		    "Available": true
    		  }
    		}
  		]
}
`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Fatal("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Fatal(err)
		}
		tuStub := TaskUpdate{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != f.Rooms[0].Id {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, f.Rooms[0].Id)
		}
	})

	t.Run("should reassign with correct order when done", func(t *testing.T) {
		var floorStub = `{
  		"FloorName": "Awesome floor",
  		"Tasks": [
  		  {
  		    "Id": "0",
  		    "Name": "Küche reinigen",
  		    "AssignedTo": 0,
  		    "Reminders": 1,
  		    "AssignmentDate": "2024-06-13T14:48:00.000Z"
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
  		    "Order": 2,
  		    "Resident": {
  		      "Id": "2",
  		      "Name": "Leona Musterman",
  		      "Available": true
  		    }
  		  },
    		{
    		  "Id": 2,
    		  "Number": "303",
    		  "Order": 4,
    		  "Resident": {
    		    "Id": "3",
    		    "Name": "Evelyn Weber",
    		    "Available": true
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
    		  "Order": 1,
    		  "Resident": {
    		    "Id": "5",
    		    "Name": "Benjamin Renert",
    		    "Available": true
    		  }
				}
  		]
}
`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Fatal("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Fatal(err)
		}
		tuStub := TaskUpdate{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdate{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != f.Rooms[4].Id {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, f.Rooms[4].Id)
		}
	})
}

func insertTestFloor(f Floor) (Floor, error) {
	floor, err := insertNewFloor(f)
	if err != nil {
		return Floor{}, err
	}
	floorsCreated = append(floorsCreated, floor.Id)
	return floor, nil
}
