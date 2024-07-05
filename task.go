package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type TaskService interface {
	handleTaskUpdate(floorId string, task Task, action string, resident Resident) error
}

type TaskUpdate struct {
	FloorId string `json:"floorId"`
	Task    Task   `json:"task"`
	Action  string `json:"action"`
	Room    Room   `json:"room"`
}

// TODO replace with ok
func findTask(tasks []Task, taskID string) (Task, error) {
	for _, t := range tasks {
		if t.Id == taskID {
			return t
		}
	}
	return Task{}, fmt.Errorf("Task not found")
}

func isAssigneeConsistent(f Floor, t Task) (bool, error) {
	task := findTask(f.Tasks, t.Id)

	if err != nil {
		return false, err
	}

	return fromDB.AssignedTo == fromUser.AssignedTo
}

func nextAssignee(f Floor, t Task) (Room, error) {
	var currentAss Room
	var roomFound bool
	for _, r := range f.Rooms {
		if r.Id == t.AssignedTo {
			currentAss = r
			roomFound = true
			break
		}
	}
	if !roomFound {
		return Room{}, fmt.Errorf("Room not found")
	}
	nextAss := currentAss
	for {
		nextOrder := (nextAss.Order + 1) % len(f.Rooms)
		for _, r := range f.Rooms {
			if r.Order == nextOrder {
				nextAss = r
				break
			}
		}
		if nextAss.Id == currentAss.Id { //looped through all rooms => break from inf. loop
			break
		}
		if nextAss.Resident.Available == true {
			break
		}
	}

	if nextAss.Id == currentAss.Id {
		return Room{}, fmt.Errorf("No next assignee available")
	}
	return nextAss, nil
}

func assignTask(t Task, r Room) Task {
	t.AssignedTo = r.Id
	t.AssignmentDate = time.Now()
	t.Reminders = 0
	return t
}

func (s TaskUpdate) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	var taskUpdate TaskUpdate
	err := json.NewDecoder(r.Body).Decode(&taskUpdate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	floor, err := getFloor(taskUpdate.FloorId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Floor not found", http.StatusNotFound)
			return
		}
	}
	isAssigneeConsistent(floor, taskUpdate.Task)
}
