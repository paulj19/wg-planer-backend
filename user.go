package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

func HandleAvailabilityStatusChange(w http.ResponseWriter, r *http.Request) {
	floorId := "669fca69d244526d709f6d76"
	var userId = "2"
	corsHandler(w)
	if r.Method == http.MethodOptions {
		return
	}
	var taskUpdate TaskUpdate
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

	if taskUpdate.Action == "RESIDENT_AVAILABLE" {
		floor.Rooms[roomIndex].Resident.Available = true
	} else if taskUpdate.Action == "RESIDENT_UNAVAILABLE" {
		floor, err = processTaskUpdate(&floor, taskUpdate)
		if err != nil {
			if strings.HasPrefix(err.Error(), "taskUpdate updating DB tasks:") {
				logger.Error("taskUpdate updating DB tasks", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskUpdate", taskUpdate))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.Error("taskUpdate processUpdate", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskToUpdate", taskUpdate))
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		floor.Rooms[roomIndex].Resident.Available = false
	}

	fUp, err = updateRoom(floor, roomIndex)
	if err != nil {
		logger.Error("availabilityStatusChange updating DB room", slog.Any("error", err), slog.Any("floor", floor), slog.Any("taskUpdate", taskUpdate))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fUp)
	return
}
