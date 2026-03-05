package main

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// ---------------------------------------------------------------------------
// Stub DB client
// ---------------------------------------------------------------------------

type stubDB struct {
	mu            sync.Mutex
	totalDocs     int
	wordDocCounts map[string]int
	wordDocs      map[string][]struct {
		URL string
		TF  int
	}
	writtenOps []mongo.WriteModel
}

func (s *stubDB) getDocCount(_ context.Context, word string) (int, error) {
	return s.wordDocCounts[word], nil
}

func (s *stubDB) getDocs(_ context.Context, word string) ([]struct {
	URL string
	TF  int
}, error) {
	return s.wordDocs[word], nil
}

func (s *stubDB) makeOp(word, url string, idf, tfidf float64) mongo.WriteModel {
	return mongo.NewUpdateOneModel().
		SetFilter(map[string]interface{}{"word": word, "url": url}).
		SetUpdate(map[string]interface{}{"$set": map[string]interface{}{"idf": idf, "tfidf": tfidf}}).
		SetUpsert(true)
}

func (s *stubDB) bulkWrite(_ context.Context, ops []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writtenOps = append(s.writtenOps, ops...)
	return &mongo.BulkWriteResult{ModifiedCount: int64(len(ops))}, nil
}

func (s *stubDB) opCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.writtenOps)
}

// ---------------------------------------------------------------------------
// workerLogic mirrors the real worker with injected functions so it can be
// unit-tested without a live MongoDB instance.
// ---------------------------------------------------------------------------

func workerLogic(
	ctx context.Context,
	id int,
	wordChan <-chan string,
	totalDocs int,
	getDocCount func(context.Context, string) (int, error),
	getDocs func(context.Context, string) ([]struct {
		URL string
		TF  int
	}, error),
	makeOp func(string, string, float64, float64) mongo.WriteModel,
	opsChan chan<- mongo.WriteModel,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	for word := range wordChan {
		docCount, err := getDocCount(ctx, word)
		if err != nil || docCount == 0 {
			continue
		}
		idf := math.Log10(float64(totalDocs) / float64(1+docCount))
		docs, err := getDocs(ctx, word)
		if err != nil {
			continue
		}
		for _, doc := range docs {
			tfidf := float64(doc.TF) * idf
			opsChan <- makeOp(word, doc.URL, idf, tfidf)
		}
	}
}

// ---------------------------------------------------------------------------
// bulkWriterLogic mirrors the real bulkWriter with an injected write func.
// ---------------------------------------------------------------------------

func bulkWriterLogic(
	ctx context.Context,
	bulkWrite func(context.Context, []mongo.WriteModel) (*mongo.BulkWriteResult, error),
	opsChan <-chan mongo.WriteModel,
	threshold int,
	done <-chan struct{},
) {
	var ops []mongo.WriteModel
	for {
		select {
		case op := <-opsChan:
			ops = append(ops, op)
			if len(ops) >= threshold {
				_, _ = bulkWrite(ctx, ops)
				ops = ops[:0]
			}
		case <-done:
			if len(ops) > 0 {
				_, _ = bulkWrite(ctx, ops)
			}
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func drainOps(ch <-chan mongo.WriteModel) []mongo.WriteModel {
	var out []mongo.WriteModel
	for op := range ch {
		out = append(out, op)
	}
	return out
}

func dummyOp(i int) mongo.WriteModel {
	return mongo.NewUpdateOneModel().
		SetFilter(map[string]interface{}{"i": i}).
		SetUpdate(map[string]interface{}{"$set": map[string]interface{}{"v": i}})
}

// ---------------------------------------------------------------------------
// Tests: workerLogic
// ---------------------------------------------------------------------------

// A word with zero documents in the index must produce no write ops.
func TestWorker_SkipsWordWithZeroDocCount(t *testing.T) {
	db := &stubDB{
		totalDocs:     100,
		wordDocCounts: map[string]int{"ghost": 0},
		wordDocs: map[string][]struct {
			URL string
			TF  int
		}{},
	}

	wordChan := make(chan string, 1)
	opsChan := make(chan mongo.WriteModel, 10)
	wordChan <- "ghost"
	close(wordChan)

	var wg sync.WaitGroup
	wg.Add(1)
	go workerLogic(context.Background(), 1, wordChan, db.totalDocs,
		db.getDocCount, db.getDocs, db.makeOp, opsChan, &wg)
	wg.Wait()
	close(opsChan)

	if ops := drainOps(opsChan); len(ops) != 0 {
		t.Errorf("expected 0 ops for a word with zero doc count, got %d", len(ops))
	}
}

// A word present in N documents must produce exactly N write ops.
func TestWorker_ProducesOneOpPerDocument(t *testing.T) {
	db := &stubDB{
		totalDocs:     100,
		wordDocCounts: map[string]int{"golang": 10},
		wordDocs: map[string][]struct {
			URL string
			TF  int
		}{
			"golang": {
				{URL: "http://a.com", TF: 3},
				{URL: "http://b.com", TF: 5},
				{URL: "http://c.com", TF: 1},
			},
		},
	}

	wordChan := make(chan string, 1)
	opsChan := make(chan mongo.WriteModel, 20)
	wordChan <- "golang"
	close(wordChan)

	var wg sync.WaitGroup
	wg.Add(1)
	go workerLogic(context.Background(), 1, wordChan, db.totalDocs,
		db.getDocCount, db.getDocs, db.makeOp, opsChan, &wg)
	wg.Wait()
	close(opsChan)

	if ops := drainOps(opsChan); len(ops) != 3 {
		t.Errorf("expected 3 ops, got %d", len(ops))
	}
}

// TF-IDF value must equal tf * log10(totalDocs / (1 + docCount)).
func TestWorker_TFIDFCalculationIsCorrect(t *testing.T) {
	// Choose values so IDF = log10(1000/100) = log10(10) = 1.0
	totalDocs := 1000
	docCount := 99
	tf := 4

	var capturedIDF, capturedTFIDF float64
	captureOp := func(word, url string, idf, tfidf float64) mongo.WriteModel {
		capturedIDF = idf
		capturedTFIDF = tfidf
		return mongo.NewUpdateOneModel().
			SetFilter(map[string]interface{}{}).
			SetUpdate(map[string]interface{}{})
	}

	db := &stubDB{
		totalDocs:     totalDocs,
		wordDocCounts: map[string]int{"word": docCount},
		wordDocs: map[string][]struct {
			URL string
			TF  int
		}{
			"word": {{URL: "http://x.com", TF: tf}},
		},
	}

	wordChan := make(chan string, 1)
	opsChan := make(chan mongo.WriteModel, 10)
	wordChan <- "word"
	close(wordChan)

	var wg sync.WaitGroup
	wg.Add(1)
	go workerLogic(context.Background(), 1, wordChan, db.totalDocs,
		db.getDocCount, db.getDocs, captureOp, opsChan, &wg)
	wg.Wait()
	close(opsChan)
	drainOps(opsChan)

	expectedIDF := math.Log10(float64(totalDocs) / float64(1+docCount))
	expectedTFIDF := float64(tf) * expectedIDF
	const eps = 1e-9
	if math.Abs(capturedIDF-expectedIDF) > eps {
		t.Errorf("IDF: want %.6f, got %.6f", expectedIDF, capturedIDF)
	}
	if math.Abs(capturedTFIDF-expectedTFIDF) > eps {
		t.Errorf("TF-IDF: want %.6f, got %.6f", expectedTFIDF, capturedTFIDF)
	}
}

// Multiple concurrent workers must collectively produce one op per (word, document) pair.
func TestWorker_ConcurrentWorkers_ProcessAllWords(t *testing.T) {
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	docCounts := map[string]int{}
	wordDocs := map[string][]struct {
		URL string
		TF  int
	}{}
	for _, w := range words {
		docCounts[w] = 5
		wordDocs[w] = []struct {
			URL string
			TF  int
		}{{URL: "http://x.com", TF: 2}}
	}
	db := &stubDB{totalDocs: 50, wordDocCounts: docCounts, wordDocs: wordDocs}

	wordChan := make(chan string, len(words))
	opsChan := make(chan mongo.WriteModel, 100)
	for _, w := range words {
		wordChan <- w
	}
	close(wordChan)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go workerLogic(context.Background(), i+1, wordChan, db.totalDocs,
			db.getDocCount, db.getDocs, db.makeOp, opsChan, &wg)
	}
	wg.Wait()
	close(opsChan)

	// Each word has exactly 1 document → expect len(words) total ops.
	if ops := drainOps(opsChan); len(ops) != len(words) {
		t.Errorf("expected %d ops, got %d", len(words), len(ops))
	}
}

// ---------------------------------------------------------------------------
// Tests: bulkWriterLogic
// ---------------------------------------------------------------------------

// When the threshold is reached the writer must flush immediately.
func TestBulkWriter_FlushesAtThreshold(t *testing.T) {
	db := &stubDB{}
	opsChan := make(chan mongo.WriteModel, 200)
	done := make(chan struct{})

	go bulkWriterLogic(context.Background(), db.bulkWrite, opsChan, 5, done)

	for i := 0; i < 10; i++ {
		opsChan <- dummyOp(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(done)
	time.Sleep(30 * time.Millisecond)

	if got := db.opCount(); got != 10 {
		t.Errorf("expected 10 written ops, got %d", got)
	}
}

// Ops below threshold must be flushed when done is closed.
func TestBulkWriter_FlushesRemainingOnDone(t *testing.T) {
	db := &stubDB{}
	opsChan := make(chan mongo.WriteModel, 50)
	done := make(chan struct{})

	go bulkWriterLogic(context.Background(), db.bulkWrite, opsChan, 100, done)

	for i := 0; i < 7; i++ {
		opsChan <- dummyOp(i)
	}
	time.Sleep(20 * time.Millisecond)
	close(done)
	time.Sleep(30 * time.Millisecond)

	if got := db.opCount(); got != 7 {
		t.Errorf("expected 7 ops flushed on done, got %d", got)
	}
}

// An empty channel followed by done must result in zero bulk-write calls.
func TestBulkWriter_NoOpsOnDone_DoesNotCallBulkWrite(t *testing.T) {
	db := &stubDB{}
	opsChan := make(chan mongo.WriteModel, 10)
	done := make(chan struct{})

	go bulkWriterLogic(context.Background(), db.bulkWrite, opsChan, 5, done)

	close(done)
	time.Sleep(30 * time.Millisecond)

	if got := db.opCount(); got != 0 {
		t.Errorf("expected 0 ops, got %d", got)
	}
}
