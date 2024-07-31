package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type CodeGenRequest struct {
	Room Room `json:"room"`
}

type CodeMapEntry struct {
	FloorId string `json:"floorId"`
	Room    Room   `json:"room"`
}

type CodeSubmitResponse struct {
	Floor Floor `json:"floor"`
	Room  Room  `json:"room"`
}

type CodeGenResponse struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

var codeMap = make(map[string]CodeMapEntry)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

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

func HandleCodeGeneration(w http.ResponseWriter, r *http.Request) {
	corsHandler(w)
	floorId := "669fca69d244526d709f6d76"
	if r.Method == http.MethodOptions {
		return
	}
	var args CodeGenRequest
	err := json.NewDecoder(r.Body).Decode(&args)
	if err != nil {
		logger.Error("codeGeneration decoding data payload", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	code := generateCode()
	codeGenResponse := CodeGenResponse{
		Code:      code,
		Timestamp: time.Now(),
	}
	codeMap[code] = CodeMapEntry{
		FloorId: floorId,
		Room:    args.Room,
	}
	time.AfterFunc(10*time.Second, func() {
		delete(codeMap, code)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(codeGenResponse)
}

func HandleCodeSubmit(w http.ResponseWriter, r *http.Request) {
	floorId := "669fca69d244526d709f6d76"
	corsHandler(w)
	if r.Method == http.MethodOptions {
		return
	}
	var code string
	err := json.NewDecoder(r.Body).Decode(&code)
	if err != nil {
		logger.Error("codeSubmit decoding data payload", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	args, ok := codeMap[code]
	if !ok {
		http.Error(w, "Code not found", http.StatusUnprocessableEntity)
		return
	}
	floor, err := getFloor(floorId)
	if err != nil {
		logger.Error("codeSubmit getFloor", slog.Any("error", err), slog.Any("args", args))
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Floor not found", http.StatusUnprocessableEntity)
			return
		}
	}

	roomIndex, err := findRoomById(floor.Rooms, args.Room.Id)
	if err != nil {
		logger.Error("codeSubmit findRoom", slog.Any("error", err), slog.Any("floor", floor), slog.Any("args", args))
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	//consistency check
	if !reflect.DeepEqual(floor.Rooms[roomIndex], args.Room) {
		http.Error(w, "Room changed since code generation", http.StatusUnprocessableEntity)
		return
	}

	delete(codeMap, code)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CodeSubmitResponse{Floor: floor, Room: args.Room})
}

func generateCode() string {
	code := make([]byte, 4)
	const samples = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < 4; i++ {
		code[i] = samples[r.Intn(len(samples))]
	}
	return string(code)
}
