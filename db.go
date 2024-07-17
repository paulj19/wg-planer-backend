package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

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

func insertNewFloor(floor Floor) (Floor, error) {
	res, err := collection.InsertOne(context.Background(), floor)
	if err != nil {
		return Floor{}, err
	}
	// insertedID, ok := res.InsertedID.(primitive.ObjectID)
	// if !ok {
	// 	return "", fmt.Errorf("newly inserted id could not be retrieved")
	// }
	// return insertedID.Hex(), nil
	var newFloor Floor
	err = collection.FindOne(context.Background(), bson.M{"_id": res.InsertedID}).Decode(&newFloor)
	if err != nil {
		return Floor{}, fmt.Errorf("newly inserted floor could not be retrieved %w", err)
	}
	log.Println("added new floor: ", newFloor)
	return newFloor, nil
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

func deleteTestFloors(fIds []primitive.ObjectID) {
	_, err := collection.DeleteMany(context.Background(), bson.M{"_id": bson.M{"$in": fIds}})
	if err != nil {
		log.Fatal(err)
	}
}

func updateTask(f Floor, taskIndex int) (Floor, error) {
	var fUpdated Floor
	result, err := collection.UpdateOne(context.Background(), bson.M{"_id": f.Id}, bson.M{"$set": bson.M{"tasks." + strconv.Itoa(taskIndex): f.Tasks[taskIndex]}})
	if err != nil {
		return Floor{}, err
	}
	if result.ModifiedCount == 0 {
		return Floor{}, mongo.ErrNoDocuments
	}
	err = collection.FindOne(context.Background(), bson.M{"_id": f.Id}).Decode(&fUpdated)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}
