package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

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
			t.Error(err)
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
			t.Error(err)
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
		t.Error(err)
	}
	t.Run("should assign task", func(t *testing.T) {
		tuStub := TaskUpdateRequest{
			FloorId:  f.Id.String()[10:34],
			Task:     FloorStub.Tasks[0],
			Action:   "ASSIGN",
			NextRoom: FloorStub.Rooms[3],
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != FloorStub.Rooms[3].Id {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, FloorStub.Rooms[2].Id)
		}
	})

	t.Run("should 422 when resident unavailable after user click", func(t *testing.T) {
		FloorStub.Rooms[3].Resident.Available = false
		tuStub := TaskUpdateRequest{
			FloorId:  f.Id.String()[10:34],
			Task:     FloorStub.Tasks[0],
			Action:   "ASSIGN",
			NextRoom: FloorStub.Rooms[3],
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnprocessableEntity)
		}
	})
	t.Run("should return 422 due to ass unavail", func(t *testing.T) {
		tuStub := TaskUpdateRequest{
			FloorId:  f.Id.String()[10:34],
			Task:     FloorStub.Tasks[0],
			Action:   "ASSIGN",
			NextRoom: FloorStub.Rooms[2],
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnprocessableEntity)
		}
	})

	t.Run("should UNASSIGN task", func(t *testing.T) {
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "UNASSIGN",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != -1 {
			t.Errorf("task not assigned correctly: got %v want %v", updatedFloor.Tasks[0].AssignedTo, -1)
		}
	})

	t.Run("should reassign task DONE", func(t *testing.T) {
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
			t.Error("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		floorsCreated = append(floorsCreated, f.Id)
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

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
			t.Error("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
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
			t.Error("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
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
}`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Error("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
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
			t.Error("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
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
}`
		err := json.Unmarshal([]byte(floorStub), &FloorStub)
		if err != nil {
			t.Error("TestSetUp could not unmarshal FloorStub ", err)
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "DONE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/task-update", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskUpdate)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].AssignedTo != f.Rooms[4].Id {
			t.Errorf("task not assigned: got %v want %v", updatedFloor.Tasks[0].AssignedTo, f.Rooms[4].Id)
		}
	})
}

func Test_remindTask(t *testing.T) {
	t.Run("should remind task", func(t *testing.T) {
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Task:    FloorStub.Tasks[0],
			Action:  "REMIND",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/remind-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		services := services{taskService: TaskUpdateRequest{}}
		handler := http.HandlerFunc(services.taskService.HandleTaskRemind)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Tasks[0].Reminders != 2 {
			t.Errorf("task not reminded: got %v want %v", updatedFloor.Tasks[0].Reminders, 2)
		}
	})
}

func Test_residentUnavailable(t *testing.T) {
	t.Run("should pass all tasks of RESIDENT_UNAVAILABLE", func(t *testing.T) {
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Action:  "RESIDENT_UNAVAILABLE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/update-availability", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleAvailabilityStatusChange)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Rooms[1].Resident.Available != false {
			t.Errorf("resident not unavailable: got %v want %v", updatedFloor.Rooms[0].Resident.Available, false)
		}
		for i := 1; i < len(updatedFloor.Tasks); i++ {
			if updatedFloor.Tasks[i].AssignedTo != 3 {
				t.Errorf("task not assigned correctly: got %v want %v", updatedFloor.Tasks[i].AssignedTo, 2)
			}
		}
	})
	t.Run("should unassign all tasks RESIDENT_UNAVAILABLE", func(t *testing.T) {
		for i := 0; i < len(FloorStub.Rooms); i++ {
			if FloorStub.Rooms[i].Id != 1 {
				FloorStub.Rooms[i].Resident.Available = false
			}
		}
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}

		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Action:  "RESIDENT_UNAVAILABLE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/update-availability", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleAvailabilityStatusChange)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Rooms[1].Resident.Available != false {
			t.Errorf("resident not unavailable: got %v want %v", updatedFloor.Rooms[0].Resident.Available, false)
		}
		for i := 1; i < len(updatedFloor.Tasks); i++ {
			if updatedFloor.Tasks[i].AssignedTo != -1 {
				t.Errorf("task not assigned correctly: got %v want %v", updatedFloor.Tasks[i].AssignedTo, -1)
			}
		}
	})
	t.Run("should set resident available", func(t *testing.T) {
		FloorStub.Rooms[1].Resident.Available = false
		f, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}

		tuStub := TaskUpdateRequest{
			FloorId: f.Id.String()[10:34],
			Action:  "RESIDENT_AVAILABLE",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/update-availability", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleAvailabilityStatusChange)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		if updatedFloor.Rooms[1].Resident.Available != true {
			t.Errorf("resident not unavailable: got %v want %v", updatedFloor.Rooms[0].Resident.Available, false)
		}
	})
}

func Test_createTask(t *testing.T) {
	t.Run("should create voting", func(t *testing.T) {
		randomTaskName := strconv.Itoa(rand.Intn(100)) + " new task"
		_, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskVotingRequest{
			Task:   Task{Name: randomTaskName},
			Action: "CREATE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:           1,
			Type:         "CREATE_TASK",
			Data:         Task{Name: randomTaskName},
			VotingWindow: 2 * 24 * time.Hour,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

	})
	t.Run("should delete voting on timeout", func(t *testing.T) {
		randomTaskName := strconv.Itoa(rand.Intn(100)) + " new task"
		tuStub := TaskVotingRequest{
			Task:   Task{Name: randomTaskName},
			Action: "CREATE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:   1,
			Type: "CREATE_TASK",
			Data: Task{Name: randomTaskName},
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		time.Sleep(12 * time.Second)

		fId, err := primitive.ObjectIDFromHex("669fca69d244526d709f6d76")
		if err != nil {
			t.Error(err)
		}
		_, err = FindVoting(fId, updatedFloor.Votings[0].Id)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("voting not deleted: got %v want %v", err, nil)
		}
	})

	// t.Run("should increase accept count", func(t *testing.T) {
	// 	_, err := insertTestFloor(FloorStub)
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// 	tuStub := CreateTaskRequest{
	// 		Taskname: "Test Task",
	// 	}
	// 	tuStubStr, err := json.Marshal(tuStub)
	// 	req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// 	rr := httptest.NewRecorder()
	// 	handler := http.HandlerFunc(HandleCreateTask)
	// 	handler.ServeHTTP(rr, req)

	// 	if status := rr.Code; status != http.StatusCreated {
	// 		t.Errorf("handler returned wrong status code: got %v want %v",
	// 			status, http.StatusCreated)
	// 	}

	// 	var updatedFloor Floor
	// 	json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

	// 	expectedVoting := Voting{
	// 		Id:           1,
	// 		Type:         "CREATE_TASK",
	// 		Data:         "Test Task",
	// 		VotingWindow: 2 * 24 * time.Hour,
	// 	}

	// 	if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].Accepts != expectedVoting.Accepts && updatedFloor.Votings[0].Rejects != expectedVoting.Rejects && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {

	// 		t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
	// 	}

	// 	votingAccept := VotingRequest{
	// 		Voting: updatedFloor.Votings[0],
	// 		Action: "ACCEPT",
	// 	}

	// 	votingAcceptStr, err := json.Marshal(votingAccept)
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// 	req, err = http.NewRequest("POST", "/update-voting", bytes.NewReader(votingAcceptStr))
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// 	rr = httptest.NewRecorder()
	// 	handler = http.HandlerFunc(HandleAcceptTaskCreate)
	// 	handler.ServeHTTP(rr, req)

	// 	if status := rr.Code; status != http.StatusOK {
	// 		t.Errorf("handler returned wrong status code: got %v want %v",
	// 			status, http.StatusOK)
	// 	}

	// 	json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

	// 	if updatedFloor.Votings[0].Accepts != 1 {
	// 		t.Errorf("voting not accepted: got %v want %v", updatedFloor.Votings[0].Accepts, 1)
	// 	}

	// 	deleteVoting(updatedFloor.Id, updatedFloor.Votings[0].Id)
	// })
	t.Run("should create task when accept", func(t *testing.T) {
		randomTaskName := strconv.Itoa(rand.Intn(100)) + " new task"
		tuStub := TaskVotingRequest{
			Task:   Task{Name: randomTaskName},
			Action: "CREATE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:           1,
			Type:         "CREATE_TASK",
			Data:         Task{Name: randomTaskName},
			VotingWindow: 2 * 24 * time.Hour,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		votingAccept := VotingActionRequest{
			Voting: updatedFloor.Votings[0],
			Action: "ACCEPT",
		}

		votingAcceptStr, err := json.Marshal(votingAccept)
		if err != nil {
			t.Error(err)
		}
		req, err = http.NewRequest("POST", "/update-voting", bytes.NewReader(votingAcceptStr))
		if err != nil {
			t.Error(err)
		}
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(HandleTaskVotingResponse)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedNewTask := Task{
			Name:       randomTaskName,
			AssignedTo: -1,
			Reminders:  0,
		}

		if updatedFloor.Tasks[len(updatedFloor.Tasks)-1].Name != expectedNewTask.Name && updatedFloor.Tasks[len(updatedFloor.Tasks)-1].AssignedTo != expectedNewTask.AssignedTo && updatedFloor.Tasks[len(updatedFloor.Tasks)-1].Reminders != expectedNewTask.Reminders {
			t.Errorf("task not created: got %v want %v", updatedFloor.Tasks[len(updatedFloor.Tasks)-1], expectedNewTask)
		}

		if len(updatedFloor.Votings) != 0 {
			t.Errorf("voting not deleted: got %v want %v", len(updatedFloor.Votings), 0)
		}
	})

	t.Run("should delete voting on reject", func(t *testing.T) {
		randomTaskName := strconv.Itoa(rand.Intn(100)) + " new task"
		_, err := insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskVotingRequest{
			Task:   Task{Name: randomTaskName},
			Action: "CREATE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:           1,
			Type:         "CREATE_TASK",
			Data:         Task{Name: randomTaskName},
			VotingWindow: 2 * 24 * time.Hour,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		votingAccept := VotingActionRequest{
			Voting: updatedFloor.Votings[0],
			Action: "REJECT",
		}

		votingAcceptStr, err := json.Marshal(votingAccept)
		if err != nil {
			t.Error(err)
		}
		req, err = http.NewRequest("POST", "/update-voting", bytes.NewReader(votingAcceptStr))
		if err != nil {
			t.Error(err)
		}
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(HandleTaskVotingResponse)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		for _, task := range updatedFloor.Tasks {
			if task.Name == randomTaskName {
				t.Errorf("task created on reject")
			}
		}

		if len(updatedFloor.Votings) != 0 {
			t.Errorf("voting not deleted: got %v want %v", len(updatedFloor.Votings), 0)
		}
	})
}

func Test_deleteTask(t *testing.T) {
	t.Run("should create voting", func(t *testing.T) {
		fId, err := primitive.ObjectIDFromHex("669fca69d244526d709f6d76")
		floorId = fId.String()[10:34]
		if err != nil {
			t.Error(err)
		}
		deleteAllVotings(fId)
		_, err = insertTestFloor(FloorStub)
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskVotingRequest{
			Task:   FloorStub.Tasks[0],
			Action: "DELETE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:           1,
			Type:         "DELETE_TASK",
			Data:         tuStub.Task,
			Accepts:      []string{},
			Rejects:      []string{},
			VotingWindow: 2 * 24 * time.Hour,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && len(updatedFloor.Votings[0].Accepts) != len(expectedVoting.Accepts) && len(updatedFloor.Votings[0].Rejects) != len(expectedVoting.Rejects) && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}
		deleteAllVotings(fId)
	})
	t.Run("should delete voting on timeout", func(t *testing.T) {
		tuStub := TaskVotingRequest{
			Task:   FloorStub.Tasks[0],
			Action: "DELETE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:   1,
			Type: tuStub.Action,
			Data: tuStub.Task,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		time.Sleep(12 * time.Second)

		fId, err := primitive.ObjectIDFromHex("669fca69d244526d709f6d76")
		if err != nil {
			t.Error(err)
		}
		_, err = FindVoting(fId, updatedFloor.Votings[0].Id)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("voting not deleted")
		}
	})
	t.Run("should not delete task with only one accept", func(t *testing.T) {
		tuStub := TaskVotingRequest{
			Task:   FloorStub.Tasks[0],
			Action: "DELETE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:           1,
			Type:         "DELETE_TASK",
			Data:         tuStub.Task,
			VotingWindow: 2 * 24 * time.Hour,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		votingAccept := VotingActionRequest{
			Voting: updatedFloor.Votings[0],
			Action: "ACCEPT",
		}

		votingAcceptStr, err := json.Marshal(votingAccept)
		if err != nil {
			t.Error(err)
		}
		req, err = http.NewRequest("POST", "/update-voting", bytes.NewReader(votingAcceptStr))
		if err != nil {
			t.Error(err)
		}
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(HandleTaskVotingResponse)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting = Voting{
			Id:           1,
			Type:         "DELETE_TASK",
			Data:         tuStub.Task,
			VotingWindow: 2 * 24 * time.Hour,
			Accepts:      []string{"1"},
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow && len(updatedFloor.Votings[0].Accepts) != len(expectedVoting.Accepts) && updatedFloor.Votings[0].Accepts[0] != expectedVoting.Accepts[0] {
			t.Errorf("voting not updated: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		if updatedFloor.Tasks[0].Name != FloorStub.Tasks[0].Name && updatedFloor.Tasks[0].AssignedTo != FloorStub.Tasks[0].AssignedTo && updatedFloor.Tasks[0].Reminders != FloorStub.Tasks[0].Reminders && updatedFloor.Tasks[0].Id != FloorStub.Tasks[0].Id {
			t.Errorf("task must not be updated: got %v want %v", updatedFloor.Tasks[0], FloorStub.Tasks[0])
		}
		deleteVoting(updatedFloor.Id, updatedFloor.Votings[0].Id)
	})

	t.Run("should delete task when all residents accept", func(t *testing.T) {
		//this is a special setting for one time test, will not work if the FloorStub is not inserted as userId is taken as preset in the function or will be replaced by jwt
		IsTest = true

		FloorStub.Rooms = FloorStub.Rooms[:3]
		f, err := insertTestFloor(FloorStub)
		floorId = f.Id.String()[10:34]
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskVotingRequest{
			Task:   f.Tasks[0],
			Action: "DELETE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:           1,
			Type:         "DELETE_TASK",
			Data:         tuStub.Task,
			VotingWindow: 2 * 24 * time.Hour,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		for i := 0; i < len(f.Rooms); i++ {
			userId = strconv.Itoa(i + 1)
			votingAccept := VotingActionRequest{
				Voting: updatedFloor.Votings[0],
				Action: "ACCEPT",
			}

			votingAcceptStr, err := json.Marshal(votingAccept)
			if err != nil {
				t.Error(err)
			}
			req, err = http.NewRequest("POST", "/update-voting", bytes.NewReader(votingAcceptStr))
			if err != nil {
				t.Error(err)
			}
			rr = httptest.NewRecorder()
			handler = http.HandlerFunc(HandleTaskVotingResponse)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, http.StatusOK)
			}

			json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

			expectedVoting = Voting{
				Id:           1,
				Type:         "DELETE_TASK",
				Data:         tuStub.Task,
				VotingWindow: 2 * 24 * time.Hour,
				Accepts:      []string{"1"},
			}

			if i == len(f.Rooms)-1 {
				if len(updatedFloor.Votings) != 0 {
					t.Errorf("voting not deleted: got %v want %v", len(updatedFloor.Votings), 0)
				}
				for _, ta := range updatedFloor.Tasks {
					if ta.Id == f.Tasks[0].Id {
						t.Errorf("task not deleted: got %v want %v", updatedFloor.Tasks[0], FloorStub.Tasks[1])
					}
				}
			} else {
				if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow && len(updatedFloor.Votings[0].Accepts) != i {
					t.Errorf("voting not updated: got %v want %v", updatedFloor.Votings[0], expectedVoting)
				}

				if updatedFloor.Tasks[0].Name != FloorStub.Tasks[0].Name && updatedFloor.Tasks[0].AssignedTo != FloorStub.Tasks[0].AssignedTo && updatedFloor.Tasks[0].Reminders != FloorStub.Tasks[0].Reminders && updatedFloor.Tasks[0].Id != FloorStub.Tasks[0].Id {
					t.Errorf("task must not be updated: got %v want %v", updatedFloor.Tasks[0], FloorStub.Tasks[0])
				}
			}
		}
	})
	t.Run("should delete voting on reject", func(t *testing.T) {
		f, err := insertTestFloor(FloorStub)
		floorId = f.Id.String()[10:34]
		if err != nil {
			t.Error(err)
		}
		tuStub := TaskVotingRequest{
			Task:   f.Tasks[0],
			Action: "DELETE_TASK",
		}
		tuStubStr, err := json.Marshal(tuStub)
		req, err := http.NewRequest("POST", "/create-del-task", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleTaskCreateDelete)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		var updatedFloor Floor
		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		expectedVoting := Voting{
			Id:           1,
			Type:         "DELETE_TASK",
			Data:         tuStub.Task,
			VotingWindow: 2 * 24 * time.Hour,
		}

		if updatedFloor.Votings[0].Type != expectedVoting.Type && updatedFloor.Votings[0].Data != expectedVoting.Data && updatedFloor.Votings[0].VotingWindow != expectedVoting.VotingWindow {
			t.Errorf("voting not created: got %v want %v", updatedFloor.Votings[0], expectedVoting)
		}

		votingAccept := VotingActionRequest{
			Voting: updatedFloor.Votings[0],
			Action: "REJECT",
		}

		votingAcceptStr, err := json.Marshal(votingAccept)
		if err != nil {
			t.Error(err)
		}
		req, err = http.NewRequest("POST", "/update-voting", bytes.NewReader(votingAcceptStr))
		if err != nil {
			t.Error(err)
		}
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(HandleTaskVotingResponse)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		json.Unmarshal(rr.Body.Bytes(), &updatedFloor)

		taskFound := false
		for _, task := range updatedFloor.Tasks {
			if task.Id == f.Tasks[0].Id {
				taskFound = true
				break
			}
		}

		if !taskFound {
			t.Errorf("Task %v must not be deleted on reject", f.Tasks[0])
		}

		if len(updatedFloor.Votings) != 0 {
			t.Errorf("voting not deleted: got %v want %v", len(updatedFloor.Votings), 0)
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
