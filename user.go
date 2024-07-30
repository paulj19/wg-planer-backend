package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

func HandleAvailabilityStatusChange(w http.ResponseWriter, r *http.Request) {
	floorId := "669fca69d244526d709f6d76"
	var userId = "1"
	if r.UserAgent() == "okhttp/4.9.2" {
		userId = "2"
	} else {
		userId = "1"
	}
	corsHandler(w)
	if r.Method == http.MethodOptions {
		return
	}
	var taskUpdate TaskUpdateRequest
	err := json.NewDecoder(r.Body).Decode(&taskUpdate)
	if err != nil {
		logger.Error("availabilityStatusChange decoding data payload", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	floor, err := getFloor(floorId)
	if err != nil {
		logger.Error("availabilityStatusChange getFloor", slog.Any("error", err), slog.Any("taskUpdate", taskUpdate))
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Floor not found", http.StatusUnprocessableEntity)
			return
		}
	}
	var fUp Floor

	roomIndex, err := findRoom(floor.Rooms, userId)

	if err != nil {
		logger.Error("taskUpdate findRoom", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate))
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	var taskUpdateResult TaskUpdateResult
	if taskUpdate.Action == "RESIDENT_AVAILABLE" {
		floor.Rooms[roomIndex].Resident.Available = true
	} else if taskUpdate.Action == "RESIDENT_UNAVAILABLE" {
		taskUpdateResult, err = processTaskUpdate(&floor, taskUpdate)
		if err != nil {
			if strings.HasPrefix(err.Error(), "taskUpdate updating DB tasks:") {
				logger.Error("taskUpdate updating DB tasks", slog.Any("error", err), slog.Any("floor", taskUpdateResult.Floor), slog.Any("taskUpdate", taskUpdateResult.TasksUpdated))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.Error("taskUpdate processUpdate", slog.Any("error", err), slog.Any("floor", taskUpdateResult.Floor), slog.Any("taskToUpdate", taskUpdateResult.TasksUpdated))
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		taskUpdateResult.Floor.Rooms[roomIndex].Resident.Available = false
		floor = taskUpdateResult.Floor
	}

	fUp, err = updateRoom(floor, roomIndex)
	if err != nil {
		logger.Error("availabilityStatusChange updating DB room", slog.Any("error", err), slog.Any("floor", taskUpdateResult.Floor), slog.Any("taskUpdate", taskUpdateResult.TasksUpdated))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fUp)

	//todo pointer check
	if !reflect.DeepEqual(taskUpdateResult.RoomToNotify, Room{}) {
		tasksJSON, err := json.Marshal(taskUpdateResult.Floor.Tasks)
		if err != nil {
			logger.Error("taskUpdate marshalling task to json", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var taskNames []string
		for _, t := range taskUpdateResult.TasksUpdated {
			taskNames = append(taskNames, t.Name)
		}

		joinedNames := strings.Join(taskNames, ", ")
		fmt.Println("XXX", taskUpdateResult.RoomToNotify)
		for i := 0; i < 3; i++ {
			err = sendNotification(taskUpdateResult.RoomToNotify, tasksJSON, floor.Id.String()[10:len(floor.Id.String())-2], "RESIDENT_UNAVAILABLE", fmt.Sprintf("%s has been assigned to you!", joinedNames))
			if err != nil {
				logger.Error("taskUpdate sendNotification attempt: "+strconv.Itoa(i+1), slog.Any("error", err), slog.Any("floor", fUp), slog.Any("taskToUpdate", taskUpdateResult.TasksUpdated))
			} else {
				break
			}
			waitTime := 2 * time.Second << (i) // Exponential backoff with base 2
			time.Sleep(waitTime)
		}
	}
}
