package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type TaskService interface {
	HandleTaskUpdate(w http.ResponseWriter, r *http.Request)
}

type TaskUpdate struct {
	FloorId  string `json:"floorId"`
	Task     Task   `json:"task"`
	Action   string `json:"action"`
	NextRoom Room   `json:"nextRoom"`
}

// TODO replace with ok
func findTask(tasks []Task, taskID string) (Task, error) {
	for _, t := range tasks {
		if t.Id == taskID {
			return t, nil
		}
	}
	return Task{}, fmt.Errorf("Task not found")
}

func findRoom(rooms []Room, roomID int64) (Room, error) {
	for _, r := range rooms {
		if r.Id == roomID {
			return r, nil
		}
	}
	return Room{}, fmt.Errorf("Room not found")
}

func checkConsistency(f Floor, tu TaskUpdate) (bool, error) {
	taskFromDB, err := findTask(f.Tasks, tu.Task.Id)
	if err != nil {
		return false, fmt.Errorf("check consistency Error finding task. %w", err)
	}

	var roomToAssign Room
	var roomFound bool
	for _, r := range f.Rooms {
		if r.Id == tu.NextRoom.Id {
			roomToAssign = r
			roomFound = true
			break
		}
	}
	if !roomFound {
		return false, fmt.Errorf("Room not found.")
	}

	if taskFromDB.AssignedTo != tu.Task.AssignedTo {
		return false, fmt.Errorf("Task assignee changed in between")
	}
	if tu.Action == "DONE" {
		return true, nil
	}

	//check if assignee set to unavailable after user clicked, UI will show only avail residents
	if roomToAssign.Resident.Available == false {
		return false, fmt.Errorf("RoomToAssign availability changed in between")
	}

	return true, nil
}

func nextAssignee(f Floor, t Task) (Room, error) {
	var currentRoom Room
	var roomFound bool
	for _, r := range f.Rooms {
		if r.Id == t.AssignedTo {
			currentRoom = r
			roomFound = true
			break
		}
	}
	if !roomFound {
		return Room{}, fmt.Errorf("Room not found")
	}
	nextAss := currentRoom
	for {
		nextOrder := (nextAss.Order + 1) % len(f.Rooms)
		for _, r := range f.Rooms {
			if r.Order == nextOrder {
				nextAss = r
				break
			}
		}
		if nextAss.Id == currentRoom.Id { //looped through all rooms => break from inf. loop
			break
		}
		if nextAss.Resident.Available == true {
			break
		}
	}

	if nextAss.Id == currentRoom.Id {
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

func (s TaskUpdate) HandleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	var taskUpdate TaskUpdate
	err := json.NewDecoder(r.Body).Decode(&taskUpdate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	floor, err := getFloor(taskUpdate.FloorId)
	if err != nil {
		logger.Error("taskUpdate getFloor", slog.Any("error", err), slog.Any("taskToUpdate", taskUpdate.Task))
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Floor not found", http.StatusUnprocessableEntity)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")

	isConsistent, err := checkConsistency(floor, taskUpdate)

	if err != nil || !isConsistent {
		logger.Error("taskUpdate checkConsistency", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate.Task))
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	nextRoom := taskUpdate.NextRoom
	if taskUpdate.Action == "DONE" {
		nextRoom, err = nextAssignee(floor, taskUpdate.Task)
		if err != nil {
			logger.Error("taskUpdate nextAssignee", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate.Task))
			if err.Error() == "No next assignee available" {
				taskUpdate.Task.AssignedTo = -1
				taskUpdate.Task.AssignmentDate = time.Now()
				taskUpdate.Task.Reminders = 0
				updateDB(floor, taskUpdate.Task)
				json.NewEncoder(w).Encode(floor)
				return
			}
			json.NewEncoder(w).Encode(floor)
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
	}

	floor, err = updateDB(floor, assignTask(taskUpdate.Task, nextRoom))
	if err != nil {
		logger.Error("taskUpdate updating DB", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate.Task))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(floor)
}
