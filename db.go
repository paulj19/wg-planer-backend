package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var client *mongo.Client
// var DB_URI = "mongodb://localhost:27018"
// var DB_URI = os.Getenv("MONGO_URL")
var DB_URI = "mongodb://wg-planer:1a865dab20bcbdd2a6015d4b81bb594d@172.17.0.6:27017/wg_planer"

func initMongo(ctx context.Context) {
// 	credential := options.Credential{
// 		AuthMechanism: "SCRAM-SHA-256",
// 		AuthSource:    "admin",
// 		Username:      "goBE_mongodb",
// 		Password:      "361c61dab61a9ed9fa598ea42c89d9e2",
// 	}
	var err error
	log.Println("connecting to db: ", DB_URI)
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(DB_URI))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	collection = client.Database("wg_planer").Collection("floor")
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

func FindFloorByUserID(rId string) (Floor, error) {
	var floor Floor
	err := collection.FindOne(context.Background(), bson.M{"rooms.resident.id": rId}).Decode(&floor)
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

func deleteTask(fId primitive.ObjectID, taskId string) (Floor, error) {
	_, err := collection.UpdateOne(context.Background(),
		bson.M{"_id": fId},
		bson.M{"$pull": bson.M{"tasks": bson.M{"id": taskId}}})
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

func FindVotingByUserID(userID string, votingID int) (Voting, error) {
	var voting Voting
	filter := bson.M{
		"$and": []bson.M{
			{"rooms.resident.id": userID},
			{"votings.id": votingID},
		},
	}
	err := collection.FindOne(context.Background(), filter).Decode(&voting)
	if err != nil {
		return Voting{}, err
	}
	return voting, nil
}
func FindVoting(fId primitive.ObjectID, votingId int) (Voting, error) {
	var voting Voting
	//TODO make it work
	// err := collection.FindOne(context.Background(),
	// 	bson.M{"_id": fId, "votings.id": votingId},
	// 	options.FindOne().SetProjection(bson.M{"votings.$": votingId})).Decode(&voting)

	var floor Floor
	err := collection.FindOne(context.Background(), bson.M{"_id": fId}).Decode(&floor)
	if err != nil {
		return Voting{}, err
	}
	for _, v := range floor.Votings {
		if v.Id == votingId {
			voting = v
			break
		}
	}

	if reflect.DeepEqual(voting, Voting{}) {
		return Voting{}, fmt.Errorf("voting with id %d not found", votingId)
	}

	return voting, nil
}

func updateVoting(fId primitive.ObjectID, voting Voting) (Floor, error) {
	//TODO remove upsert option
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
		bson.M{"$pull": bson.M{"votings": bson.M{"id": votingId}}})
	// _, err := collection.UpdateOne(context.Background(),
	// 	bson.M{"_id": fId},
	// 	bson.M{"$unset": bson.M{"votings.$[elem]": nil}},
	// 	options.Update().SetArrayFilters(options.ArrayFilters{Filters: []interface{}{bson.M{"elem.id": votingId}}}))
	if err != nil {
		return Floor{}, err
	}
	fUpdated, err := getUpdatedFloor(fId)
	if err != nil {
		return Floor{}, err
	}
	return fUpdated, nil
}

func deleteAllVotings(fId primitive.ObjectID) (Floor, error) {
	_, err := collection.UpdateOne(context.Background(),
		bson.M{"_id": fId},
		bson.M{"$unset": bson.M{"votings": []Voting{}}})
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
