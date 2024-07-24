package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	expo "github.com/oliveroneill/exponent-server-sdk-golang/sdk"
)

func sendNotification(r Room, t Task, fId string, nType string, title string) error {
	pushToken, err := expo.NewExponentPushToken(r.Resident.ExpoPushToken)
	if err != nil {
		return fmt.Errorf("error creating push token from %s: %w", r.Resident.ExpoPushToken, err)
	}

	client := expo.NewPushClient(nil)

	var m map[string]string = make(map[string]string, reflect.ValueOf(t).NumField())

	m["FloorId"] = fId[10 : len(fId)-2]
	m["Type"] = "TASK_" + nType
	taskJSON, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("error marshalling task to json: %w", err)
	}
	m["Task"] = string(taskJSON)

	pushMessage := &expo.PushMessage{
		To:       []expo.ExponentPushToken{pushToken},
		Body:     "",
		Data:     m,
		Sound:    "default",
		Title:    title,
		Priority: expo.DefaultPriority,
	}

	response, err := client.Publish(pushMessage)

	if err != nil {
		return fmt.Errorf("error publishing expo notification push message: %v, error: %w", pushMessage, err)
	}

	if response.ValidateResponse() != nil {
		return fmt.Errorf("error invalid response when sending notification reponse: %v", response)
	}
	return nil
}

// func taskToMap(t Task, m map[string]string) {
// 	val := reflect.ValueOf(t)
// 	typ := reflect.TypeOf(t)

// 	for i := 0; i < val.NumField(); i++ {
// 		m[typ.Field(i).Name] = val.Field(i).String()
// 	}
// }
