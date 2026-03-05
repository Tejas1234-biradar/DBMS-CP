package main

import (
	"context"
	"github.com/Tejas1234-biradar/DBMS-CP/src/tfi-idf/data"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	running = true
)

func worker(
	ctx context.Context,
	id int,
	wordChan <-chan string,
	totalDocs int,
	mongoClient *data.MongoClient,
	opsChan chan mongo.WriteModel,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for word := range wordChan {

		if !running {
			return
		}

		docCount, err := mongoClient.GetWordDocumentCount(ctx, word)
		if err != nil || docCount == 0 {
			continue
		}

		idf := math.Log10(float64(totalDocs) / float64(1+docCount))

		slog.Info("processing word",
			"thread", id,
			"word", word,
			"docs", docCount,
		)

		cursor, err := mongoClient.GetWordDocuments(ctx, word)
		if err != nil {
			continue
		}

		for cursor.Next(ctx) {

			var doc struct {
				URL string `bson:"url"`
				TF  int    `bson:"tf"`
			}

			if err := cursor.Decode(&doc); err != nil {
				continue
			}

			tf := float64(doc.TF)
			tfidf := tf * idf

			op := mongoClient.UpdatePageTFIDFOperation(
				word,
				doc.URL,
				idf,
				tfidf,
			)

			opsChan <- op
		}

		cursor.Close(ctx)
	}
}

func bulkWriter(
	ctx context.Context,
	mongoClient *data.MongoClient,
	opsChan <-chan mongo.WriteModel,
	threshold int,
	done chan struct{},
) {

	var operations []mongo.WriteModel

	for {

		select {

		case op := <-opsChan:

			operations = append(operations, op)

			if len(operations) >= threshold {

				slog.Info("performing bulk update", "count", len(operations))

				_, err := mongoClient.UpdatePageTFIDFBulk(ctx, operations)
				if err != nil {
					log.Println(err)
				}

				operations = operations[:0]
			}

		case <-done:

			if len(operations) > 0 {

				slog.Info("final bulk write", "count", len(operations))

				_, err := mongoClient.UpdatePageTFIDFBulk(ctx, operations)
				if err != nil {
					log.Println(err)
				}
			}

			return
		}
	}
}

func main() {

	ctx := context.Background()

	NUM_THREADS := 4
	OPERATIONS_THRESHOLD := 1000

	mongoClient, err := data.NewMongoClient(
		"localhost",
		"root",
		"password",
		"test",
		27017,
	)

	if err != nil {
		log.Fatal(err)
	}

	defer mongoClient.Close(ctx)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Warn("shutdown signal received")
		running = false
	}()

	slog.Info("fetching total documents")

	totalDocs, err := mongoClient.GetDocumentCount(ctx)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("documents found", "count", totalDocs)

	cursor, err := mongoClient.GetUniqueWords(ctx)
	if err != nil {
		log.Fatal(err)
	}

	wordChan := make(chan string, 1000)
	opsChan := make(chan mongo.WriteModel, 5000)
	done := make(chan struct{})

	// Start bulk writer
	go bulkWriter(ctx, mongoClient, opsChan, OPERATIONS_THRESHOLD, done)

	// Start workers
	var wg sync.WaitGroup

	for i := 0; i < NUM_THREADS; i++ {

		wg.Add(1)

		go worker(
			ctx,
			i+1,
			wordChan,
			totalDocs,
			mongoClient,
			opsChan,
			&wg,
		)
	}

	// Feed words to channel
	for cursor.Next(ctx) {

		var result struct {
			Word string `bson:"word"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		wordChan <- result.Word
	}

	close(wordChan)

	wg.Wait()

	close(done)

	slog.Info("TF-IDF processing complete")
}
