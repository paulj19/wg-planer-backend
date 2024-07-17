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

func (s TaskUpdate) HandleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	corsHandler(w)
	var taskUpdate TaskUpdate
	err := json.NewDecoder(r.Body).Decode(&taskUpdate)
	if err != nil {
		logger.Error("taskUpdate decoding data payload", slog.Any("error", err))
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
	taskIndex, err := findTaskIndex(floor.Tasks, taskUpdate.Task.Id)
	fmt.Println("TASK INDEX", taskIndex)
	if err != nil {
		logger.Error("taskUpdate findTaskIndex", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate.Task))
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	isConsistent, err := checkConsistency(floor, taskUpdate, taskIndex)

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
				unassignTask(&floor, taskIndex)
				floor, err := updateTask(floor, taskIndex)
				if err != nil {
					logger.Error("taskUpdate updating DB", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate.Task))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(floor)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(floor)
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
	} else if taskUpdate.Action == "UNASSIGN" {
		unassignTask(&floor, taskIndex)
		floor, err := updateTask(floor, taskIndex)
		if err != nil {
			logger.Error("taskUpdate updating DB", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate.Task))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(floor)
		return
	}

	assignTask(&floor, taskIndex, nextRoom)
	floor, err = updateTask(floor, taskIndex)
	if err != nil {
		logger.Error("taskUpdate updating DB", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate.Task))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(floor)
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

func findTaskIndex(tasks []Task, taskID string) (int, error) {
	for i, t := range tasks {
		if t.Id == taskID {
			return i, nil
		}
	}
	return -1, fmt.Errorf("Task not found")
}

func findRoom(rooms []Room, roomID int64) (Room, error) {
	for _, r := range rooms {
		if r.Id == roomID {
			return r, nil
		}
	}
	return Room{}, fmt.Errorf("Room not found")
}

func checkConsistency(f Floor, tu TaskUpdate, taskIndex int) (bool, error) {
	if f.Tasks[taskIndex].AssignedTo != tu.Task.AssignedTo {
		return false, fmt.Errorf("Task assignee changed in between")
	}
	if tu.Action == "DONE" || tu.Action == "UNASSIGN" {
		return true, nil
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

func unassignTask(f *Floor, taskIndex int) {
	f.Tasks[taskIndex].AssignedTo = -1
	f.Tasks[taskIndex].AssignmentDate = time.Now()
	f.Tasks[taskIndex].Reminders = 0
}

func assignTask(f *Floor, taskIndex int, r Room) {
	f.Tasks[taskIndex].AssignedTo = r.Id
	f.Tasks[taskIndex].AssignmentDate = time.Now()
	f.Tasks[taskIndex].Reminders = 0
}
