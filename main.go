package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Floor struct {
	Name      string   `bson:"name"`
	Residents []string `bson:"residents"`
	Tasks     []Task   `bson:"tasks"`
}

type Task struct {
	Name       string `bson:"name"`
	AssignedTo string `bson:"assignedTo"`
}

// type Resident struct {
//   Name       string `bson:"name"`
//   AssignedTo string `bson:"assignedTo"`
// }

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	initMongo(ctx)
	floorJson := []byte(`
	{
  "name": "Floor1A",
  "residents": [
    "762b569bffebb4b815cd5e78",
    "762b5ace2337d3c989bcc238",
    "762b5f46cd8a580b287a8d84"
  ],
  "tasks": [
    {
      "name": "Gelbersack entfernen",
      "assignedTo": "662b5f46cd8a580b287a8d84"
    },
    {
      "name": "Biomüll wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    },
    {
      "name": "Restmull wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    }
  ]}`)
	id, err := insertNewFloor(floorJson)
	if err != nil {
		log.Println("error inserting new floor", err)
	}
	fmt.Println(id)
	defer disconnectMongo(ctx)
}
