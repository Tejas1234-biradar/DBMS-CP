package data

import (
	"context"
	"fmt"
	"github.com/Tejas1234-biradar/DBMS-CP/src/tfi-idf/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log/slog"
)

type MongoClient struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoClient(host, username, password, dbName string, port int) (*MongoClient, error) {

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=admin",
		username,
		password,
		host,
		port,
		dbName,
	)

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	slog.Info("Successfully connected to mongo")

	return &MongoClient{
		client: client,
		db:     client.Database(dbName),
	}, nil
}

func (m *MongoClient) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
func (m *MongoClient) PerformBatchOperations(
	ctx context.Context,
	operations []mongo.WriteModel,
	collectionName string,
) (*mongo.BulkWriteResult, error) {

	if m.client == nil {
		slog.Error("Mongo connection not initialized")
		return nil, nil
	}

	if len(operations) == 0 {
		slog.Warn("No operations to perform")
		return nil, nil
	}

	res, err := m.db.Collection(collectionName).BulkWrite(
		ctx,
		operations,
		options.BulkWrite().SetOrdered(false),
	)

	if err != nil {
		return nil, fmt.Errorf("error performing batch operations: %w", err)
	}

	return res, nil
}
func (m *MongoClient) GetUniqueWords(ctx context.Context) (*mongo.Cursor, error) {

	if m.client == nil {
		slog.Error("Mongo connection not initialized")
		return nil, fmt.Errorf("mongo not initialized")
	}

	collection := m.db.Collection(utils.WORDS_COLLECTION)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$word"}}}},
		{{Key: "$project", Value: bson.D{
			{Key: "word", Value: "$_id"},
			{Key: "_id", Value: 0},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (m *MongoClient) GetWordDocumentCount(ctx context.Context, word string) (int64, error) {

	if m.client == nil {
		slog.Error("Mongo connection not initialized")
		return 0, fmt.Errorf("mongo not initialized")
	}

	collection := m.db.Collection(utils.WORDS_COLLECTION)

	count, err := collection.CountDocuments(
		ctx,
		bson.M{"word": word},
	)

	if err != nil {
		return 0, err
	}

	return count, nil
}

func (m *MongoClient) GetWordDocuments(ctx context.Context, word string) (*mongo.Cursor, error) {

	if m.client == nil {
		slog.Error("Mongo connection not initialized")
		return nil, fmt.Errorf("mongo not initialized")
	}

	collection := m.db.Collection(utils.WORDS_COLLECTION)

	cursor, err := collection.Find(
		ctx,
		bson.M{"word": word},
	)

	if err != nil {
		return nil, fmt.Errorf("failed retrieving documents for word '%s': %w", word, err)
	}

	return cursor, nil
}
func (m *MongoClient) GetDocumentCount(ctx context.Context) (int, error) {

	collection := m.db.Collection(utils.METADATA_COLLECTION)

	count, err := collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}
func (m *MongoClient) UpdatePageTFIDFOperation(
	word string,
	url string,
	idf float64,
	tfidf float64,
) mongo.WriteModel {

	filter := bson.M{
		"word": word,
		"url":  url,
	}

	update := bson.M{
		"$set": bson.M{
			"weight": tfidf,
			"idf":    idf,
		},
	}

	return mongo.NewUpdateOneModel().
		SetFilter(filter).
		SetUpdate(update)
}

func (m *MongoClient) UpdatePageTFIDFBulk(
	ctx context.Context,
	operations []mongo.WriteModel,
) (*mongo.BulkWriteResult, error) {

	if len(operations) == 0 {
		return nil, nil
	}

	return m.PerformBatchOperations(
		ctx,
		operations,
		utils.WORDS_COLLECTION,
	)
}
