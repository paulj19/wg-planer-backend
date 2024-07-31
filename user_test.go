package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func Test_codeGen(t *testing.T) {
	t.Run("should genereate code", func(t *testing.T) {
		codeStub := CodeGenRequest{
			Room: FloorStub.Rooms[6],
		}

		tuStubStr, err := json.Marshal(codeStub)
		req, err := http.NewRequest("POST", "/generate-code", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleCodeGeneration)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var resp CodeGenResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp.Code) != 4 {
			t.Errorf("expected code to be 4 characters long, got %v", len(resp.Code))
		}
	})
}

func Test_codeSubmit(t *testing.T) {
	t.Run("should submit code", func(t *testing.T) {
		codeStub := CodeGenRequest{
			Room: FloorStub.Rooms[6],
		}

		tuStubStr, err := json.Marshal(codeStub)
		req, err := http.NewRequest("POST", "/generate-code", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleCodeGeneration)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var resp CodeGenResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp.Code) != 4 {
			t.Errorf("expected code to be 4 characters long, got %v", len(resp.Code))
		}

		jsonCode, err := json.Marshal(resp.Code)
		if err != nil {
			t.Error(err)
		}
		req, err = http.NewRequest("POST", "/submit-code", bytes.NewReader(jsonCode))
		if err != nil {
			t.Error(err)
		}
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(HandleCodeSubmit)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var submitResp CodeSubmitResponse
		json.Unmarshal(rr.Body.Bytes(), &submitResp)

		if submitResp.Floor.Id.String()[10:34] != "669fca69d244526d709f6d76" {
			t.Errorf("expected floor id to be %v, got %v", "669fca69d244526d709f6d76", submitResp.Floor.Id.String()[10:34])
		}
		if !reflect.DeepEqual(submitResp.Room, FloorStub.Rooms[6]) {
			t.Errorf("expected room to be %v, got %v", FloorStub.Rooms[6], submitResp.Room)
		}
	})
	t.Run("should timeout", func(t *testing.T) {
		codeStub := CodeGenRequest{
			Room: FloorStub.Rooms[0],
		}

		tuStubStr, err := json.Marshal(codeStub)
		req, err := http.NewRequest("POST", "/generate-code", bytes.NewReader(tuStubStr))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleCodeGeneration)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var resp CodeGenResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if len(resp.Code) != 4 {
			t.Errorf("expected code to be 4 characters long, got %v", len(resp.Code))
		}

		time.Sleep(12 * time.Second)

		jsonCode, err := json.Marshal(resp.Code)
		if err != nil {
			t.Error(err)
		}
		req, err = http.NewRequest("POST", "/submit-code", bytes.NewReader(jsonCode))
		if err != nil {
			t.Error(err)
		}
		rr = httptest.NewRecorder()
		handler = http.HandlerFunc(HandleCodeSubmit)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity && rr.Body.String() != "Code not found" {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnprocessableEntity)
		}
	})
	t.Run("should return code not found when wrong code request", func(t *testing.T) {
		code := "12AB"
		jsonCode, err := json.Marshal(code)
		if err != nil {
			t.Error(err)
		}
		req, err := http.NewRequest("POST", "/submit-code", bytes.NewReader(jsonCode))
		if err != nil {
			t.Error(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HandleCodeSubmit)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity && rr.Body.String() != "Code not found" {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnprocessableEntity)
		}

	})

}
