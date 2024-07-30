package main

import (
	"fmt"

	expo "github.com/oliveroneill/exponent-server-sdk-golang/sdk"
)

func sendNotification(r Room, patch []byte, fId string, nType string, title string) error {
	pushToken, err := expo.NewExponentPushToken(r.Resident.ExpoPushToken)
	if err != nil {
		return fmt.Errorf("error creating push token from %s: %w", r.Resident.ExpoPushToken, err)
	}

	client := expo.NewPushClient(nil)

	var m map[string]string = make(map[string]string)

	m["FloorId"] = fId
	m["Type"] = nType
	m["Patch"] = string(patch)

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
