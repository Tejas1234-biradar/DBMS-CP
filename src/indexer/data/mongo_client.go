package data

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log/slog"
	"time"

	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/schemas"
)

const (
	WordsCollection      = "words"
	MetadataCollection   = "metadata"
	OutlinksCollection   = "outlinks"
	DictionaryCollection = "dictionary"
)

type MongoClient struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoClient(host, username, password, dbName string, port int) (*MongoClient, error) {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin", username, password, host, port, dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	slog.Info("Successfully connected to mongo!")

	mc := &MongoClient{
		client: client,
		db:     client.Database(dbName),
	}

	if err := mc.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return mc, nil
}

func (mc *MongoClient) createIndexes(ctx context.Context) error {
	slog.Info("Creating indexes...")
	words := mc.db.Collection(WordsCollection)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "word", Value: 1}, {Key: "url", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "word", Value: 1}, {Key: "weight", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "word", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "url", Value: 1}},
		},
	}

	_, err := words.Indexes().CreateMany(ctx, indexes)
	return err
}

func (mc *MongoClient) Close(ctx context.Context) error {
	return mc.client.Disconnect(ctx)
}

// --------------------- BATCH ---------------------

func (mc *MongoClient) performBatchOperations(ctx context.Context, collectionName string, operations []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	if len(operations) == 0 {
		slog.Warn("No operations to perform")
		return nil, nil
	}

	opts := options.BulkWrite().SetOrdered(false)
	res, err := mc.db.Collection(collectionName).BulkWrite(ctx, operations, opts)
	if err != nil {
		return nil, fmt.Errorf("bulk write error: %w", err)
	}
	return res, nil
}

// --------------------- WORDS ---------------------

func (mc *MongoClient) CreateWordsEntryOperation(word, url string, tf int) mongo.WriteModel {
	filter := bson.D{{Key: "word", Value: word}, {Key: "url", Value: url}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "tf", Value: tf},
		{Key: "weight", Value: 0},
	}}}
	return mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
}

func (mc *MongoClient) CreateWordsBulk(ctx context.Context, operations []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	return mc.performBatchOperations(ctx, WordsCollection, operations)
}

// --------------------- METADATA ---------------------

func (mc *MongoClient) GetMetadata(ctx context.Context, normalizedURL string) (*schemas.Metadata, error) {
	collection := mc.db.Collection(MetadataCollection)

	var result schemas.Metadata
	err := collection.FindOne(ctx, bson.D{{Key: "_id", Value: normalizedURL}}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	return &result, nil
}

func (mc *MongoClient) CreateMetadataEntryOperation(page *schemas.Page, htmlData *schemas.Metadata, topWords map[string]int) mongo.WriteModel {
	metadata := schemas.Metadata{
		ID:          page.NormalizedURL,
		Title:       htmlData.Title,
		Description: htmlData.Description,
		SummaryText: htmlData.SummaryText,
		LastCrawled: page.LastCrawled,
		KeyWords:    topWords,
	}

	filter := bson.D{{Key: "_id", Value: page.NormalizedURL}}
	update := bson.D{{Key: "$set", Value: metadata.ToDocument()}}
	return mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
}

func (mc *MongoClient) CreateMetadataBulk(ctx context.Context, operations []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	return mc.performBatchOperations(ctx, MetadataCollection, operations)
}

// --------------------- OUTLINKS ---------------------

func (mc *MongoClient) CreateOutlinksEntryOperation(outlinks *schemas.Outlinks) mongo.WriteModel {
	filter := bson.D{{Key: "_id", Value: outlinks.ID}}
	update := bson.D{{Key: "$set", Value: outlinks.ToDocument()}}
	return mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
}

func (mc *MongoClient) CreateOutlinksBulk(ctx context.Context, operations []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	return mc.performBatchOperations(ctx, OutlinksCollection, operations)
}

// --------------------- DICTIONARY ---------------------

func (mc *MongoClient) AddWordsToDictionary(ctx context.Context, words []string) (*mongo.BulkWriteResult, error) {
	if len(words) == 0 {
		slog.Warn("No words to add to dictionary")
		return nil, nil
	}

	operations := make([]mongo.WriteModel, len(words))
	for i, word := range words {
		filter := bson.D{{Key: "_id", Value: word}}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "_id", Value: word}}}}
		operations[i] = mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
	}

	return mc.performBatchOperations(ctx, DictionaryCollection, operations)
}
