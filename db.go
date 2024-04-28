package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var client *mongo.Client
var DB_URI = "mongodb://localhost:27018"

func initMongo(ctx context.Context) {
	credential := options.Credential{
		AuthMechanism: "SCRAM-SHA-256",
		AuthSource:    "admin",
		Username:      "wg-planer",
		Password:      "secret",
	}
	var err error
	log.Println("connecting to db: ", DB_URI)
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(DB_URI).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	collection = client.Database("wg-planer").Collection("floor")
}

func disconnectMongo(ctx context.Context) {
	if err := client.Disconnect(ctx); err != nil {
		panic(err)
	}
}

func insertNewFloor(floor Floor) (string, error) {
	log.Println("adding new floor: ", floor)
	res, err := collection.InsertOne(context.Background(), floor)
	if err != nil {
		log.Fatal(err)
	}
	insertedID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("newly inserted id could not be retrieved")
	}
	return insertedID.Hex(), nil
}

func getFloor(floorId string) (Floor, error) {
	var floor Floor
	objectId, err := primitive.ObjectIDFromHex(floorId)
	if err != nil {
		return floor, err
	}
	err = collection.FindOne(context.Background(), bson.M{"_id": objectId}).Decode(&floor)
	if err != nil {
		return floor, err
	}
	return floor, nil
}
