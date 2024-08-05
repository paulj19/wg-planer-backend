package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
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

func FindFloor(floorId string) (Floor, error) {
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

func getUpdatedFloor(fId primitive.ObjectID) (Floor, error) {
	var f Floor
	err := collection.FindOne(context.Background(), bson.M{"_id": fId}).Decode(&f)
	if err != nil {
		return Floor{}, err
	}
	return f, nil
}

func updateTasks(f Floor) (Floor, error) {
	result, err := collection.UpdateOne(context.Background(), bson.M{"_id": f.Id}, bson.M{"$set": bson.M{"tasks": f.Tasks}})
	if err != nil {
		return Floor{}, err
	}
	if result.ModifiedCount == 0 {
		return f, nil
	}
	fUpdated, err := getUpdatedFloor(f.Id)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}

func InsertTask(fId primitive.ObjectID, task Task) (Floor, error) {
	_, err := collection.UpdateOne(context.Background(),
		bson.M{"_id": fId},
		bson.M{"$push": bson.M{"tasks": task}})
	if err != nil {
		return Floor{}, err
	}

	fUpdated, err := getUpdatedFloor(fId)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}

func InsertVoting(fId primitive.ObjectID, voting Voting) (Floor, error) {
	_, err := collection.UpdateOne(context.Background(),
		bson.M{"_id": fId},
		bson.M{"$push": bson.M{"votings": voting}})
	if err != nil {
		return Floor{}, err
	}

	fUpdated, err := getUpdatedFloor(fId)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}

func FindVoting(fId primitive.ObjectID, votingId int) (Voting, error) {
	var voting Voting
	err := collection.FindOne(context.Background(),
		bson.M{"_id": fId, "votings.id": votingId}, options.FindOne().SetProjection(bson.M{"votings.$": votingId})).Decode(&voting)
	fmt.Println("XXX", voting, err)
	if err != nil {
		return Voting{}, err
	}

	if reflect.DeepEqual(voting, Voting{}) {
		return Voting{}, fmt.Errorf("voting with id %d not found", votingId)
	}

	return voting, nil
}

func updateVoting(fId primitive.ObjectID, voting Voting) (Floor, error) {
	_, err := collection.UpdateOne(context.Background(),
		bson.M{"_id": fId},
		bson.M{"$set": bson.M{"votings.$[elem]": voting}},
		options.Update().SetArrayFilters(options.ArrayFilters{Filters: []interface{}{bson.M{"elem.id": voting.Id}}}).SetUpsert(true))
	if err != nil {
		return Floor{}, err
	}

	fUpdated, err := getUpdatedFloor(fId)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}

func deleteVoting(fId primitive.ObjectID, votingId int) (Floor, error) {
	_, err := collection.UpdateOne(context.Background(),
		bson.M{"_id": fId},
		bson.M{"$unset": bson.M{"votings.$[elem]": nil}},
		options.Update().SetArrayFilters(options.ArrayFilters{Filters: []interface{}{bson.M{"elem.id": votingId}}}))
	if err != nil {
		return Floor{}, err
	}
	fUpdated, err := getUpdatedFloor(fId)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}

func updateRoom(f Floor, roomIndex int) (Floor, error) {
	result, err := collection.UpdateOne(context.Background(), bson.M{"_id": f.Id}, bson.M{"$set": bson.M{"rooms." + strconv.Itoa(roomIndex): f.Rooms[roomIndex]}})
	if err != nil {
		return Floor{}, err
	}
	if result.ModifiedCount == 0 {
		return f, nil
	}
	fUpdated, err := getUpdatedFloor(f.Id)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}

func updateExpoPushToken(f Floor, roomIndex int) (Floor, error) {
	result, err := collection.UpdateOne(context.Background(), bson.M{"_id": f.Id}, bson.M{"$set": bson.M{"rooms." + strconv.Itoa(roomIndex): f.Rooms[roomIndex]}})
	if err != nil {
		return Floor{}, fmt.Errorf("error updating expo push token in DB %w", err)
	}
	if result.ModifiedCount == 0 {
		return Floor{}, fmt.Errorf("error updating expo push token in DB, no documents modified")
	}
	fUpdated, err := getUpdatedFloor(f.Id)
	if err != nil {
		return Floor{}, fmt.Errorf("error getting updated floor from DB %w", err)
	}
	return fUpdated, nil
}
