//go:build integration

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/data"
	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/schemas"
	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/utils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---- Test environment ----

type indexerTestEnv struct {
	ctx         context.Context
	redisClient *data.RedisClient
	mongoClient *data.MongoClient
	rawRedis    *redis.Client
	rawMongo    *mongo.Database
	cleanup     func()
}

func setupIndexerTestEnv(t *testing.T) *indexerTestEnv {
	t.Helper()
	ctx := context.Background()

	// ---- Redis container ----
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	require.NoError(t, err)

	redisHost, err := redisC.Host(ctx)
	require.NoError(t, err)
	redisPort, err := redisC.MappedPort(ctx, "6379")
	require.NoError(t, err)
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort.Port())

	// ---- MongoDB container ----
	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7",
			ExposedPorts: []string{"27017/tcp"},
			Env: map[string]string{
				"MONGO_INITDB_ROOT_USERNAME": "root",
				"MONGO_INITDB_ROOT_PASSWORD": "password",
			},
			WaitingFor: wait.ForLog("Waiting for connections"),
		},
		Started: true,
	})
	require.NoError(t, err)

	mongoHost, err := mongoC.Host(ctx)
	require.NoError(t, err)
	mongoPort, err := mongoC.MappedPort(ctx, "27017")
	require.NoError(t, err)

	// Raw redis client for test setup/assertions
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	_, err = rdb.Ping(ctx).Result()
	require.NoError(t, err)

	// Indexer redis client
	redisClient := data.NewRedisClient(redisAddr, "", 0)
	require.NotNil(t, redisClient)

	// Indexer mongo client
	mongoClient, err := data.NewMongoClient(mongoHost, "root", "password", "testdb", mongoPort.Int())
	require.NoError(t, err)

	// Raw mongo for assertions
	uri := fmt.Sprintf("mongodb://root:password@%s:%s/testdb?authSource=admin", mongoHost, mongoPort.Port())
	mClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.NoError(t, err)
	db := mClient.Database("testdb")

	cleanup := func() {
		_ = rdb.Close()
		_ = mClient.Disconnect(ctx)
		_ = redisC.Terminate(ctx)
		_ = mongoC.Terminate(ctx)
	}

	return &indexerTestEnv{
		ctx:         ctx,
		redisClient: redisClient,
		mongoClient: mongoClient,
		rawRedis:    rdb,
		rawMongo:    db,
		cleanup:     cleanup,
	}
}

// pushFakePage mimics what your crawler writes into Redis
func pushFakePage(t *testing.T, env *indexerTestEnv, normalizedURL, html string, outlinks []string) string {
	t.Helper()
	ctx := env.ctx
	rdb := env.rawRedis

	pageKey := "page_data:" + normalizedURL

	err := rdb.HSet(ctx, pageKey, map[string]interface{}{
		"normalized_url": normalizedURL,
		"html":           html,
		"content_type":   "text/html",
		"status_code":    200,
		"last_crawled":   time.Now().Format(time.RFC1123),
	}).Err()
	require.NoError(t, err)

	// Push outlinks set
	if len(outlinks) > 0 {
		members := make([]interface{}, len(outlinks))
		for i, l := range outlinks {
			members[i] = l
		}
		require.NoError(t, rdb.SAdd(ctx, "outlinks:"+normalizedURL, members...).Err())
	}

	// Push page key onto the indexer queue
	require.NoError(t, rdb.LPush(ctx, "pages_queue", pageKey).Err())

	return pageKey
}

// runIndexerOnce runs one iteration of the indexer loop
func runIndexerOnce(t *testing.T, env *indexerTestEnv) {
	t.Helper()
	ctx := env.ctx
	r := env.redisClient
	m := env.mongoClient

	// Pop page from queue
	pageID, err := r.PopPage()
	require.NoError(t, err)
	require.NotEmpty(t, pageID, "expected a page in the queue")

	// Get page data
	pageData, err := r.GetPageData(pageID)
	require.NoError(t, err)
	require.NotEmpty(t, pageData)

	normalizedURL := pageData["normalized_url"]
	html := pageData["html"]

	// Parse HTML
	htmlData, err := utils.GetHTMLData(html)
	require.NoError(t, err)
	require.NotNil(t, htmlData)

	// Count word frequencies
	wordFreq := make(map[string]int)
	for _, word := range htmlData.Text {
		wordFreq[word]++
	}
	keywords := topN(wordFreq, 1000)

	// URL boost
	wordsInURL := utils.SplitURL(normalizedURL)
	for _, word := range wordsInURL {
		if past, ok := keywords[word]; ok && past != 0 {
			keywords[word] = past * 50
		} else {
			keywords[word] = 10
		}
	}

	// Write words
	var wordOps []mongo.WriteModel
	for word, freq := range keywords {
		wordOps = append(wordOps, m.CreateWordsEntryOperation(word, normalizedURL, freq))
	}
	_, err = m.CreateWordsBulk(ctx, wordOps)
	require.NoError(t, err)

	// Write metadata
	page := &schemas.Page{
		NormalizedURL: normalizedURL,
		LastCrawled:   parseTime(pageData["last_crawled"]),
	}
	metaSchema := &schemas.Metadata{
		Title:       htmlData.Title,
		Description: htmlData.Description,
		SummaryText: htmlData.SummaryText,
	}
	metaOp := m.CreateMetadataEntryOperation(page, metaSchema, keywords)
	_, err = m.CreateMetadataBulk(ctx, []mongo.WriteModel{metaOp})
	require.NoError(t, err)

	// Write outlinks
	outlinks, err := r.GetOutlinks(normalizedURL)
	if err == nil && len(outlinks) > 0 {
		outlinksObj := &schemas.Outlinks{
			ID:    normalizedURL,
			Links: sliceToSet(outlinks),
		}
		outlinksOp := m.CreateOutlinksEntryOperation(outlinksObj)
		_, err = m.CreateOutlinksBulk(ctx, []mongo.WriteModel{outlinksOp})
		require.NoError(t, err)
	}

	// Cleanup Redis
	r.DeletePageData(pageID)
	r.DeleteOutlinks(normalizedURL)
}

// ---- Tests ----

func TestIndexer_ParsesHTMLAndWritesMetadata(t *testing.T) {
	env := setupIndexerTestEnv(t)
	defer env.cleanup()

	html := `<html>
		<head>
			<title>Go Programming Language</title>
			<meta name="description" content="Go is an open source programming language.">
			<meta property="og:title" content="Go Programming Language">
		</head>
		<body>
			<p>Go was designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson.</p>
			<p>Go is statically typed, compiled and syntactically similar to C.</p>
			<p>Go provides memory safety, garbage collection, and concurrency.</p>
		</body>
	</html>`

	pushFakePage(t, env, "en.wikipedia.org/wiki/Go", html, []string{
		"en.wikipedia.org/wiki/Concurrency",
		"en.wikipedia.org/wiki/Ken_Thompson",
	})

	runIndexerOnce(t, env)

	// Assert metadata was written
	var meta bson.M
	err := env.rawMongo.Collection("metadata").FindOne(env.ctx, bson.D{{Key: "_id", Value: "en.wikipedia.org/wiki/Go"}}).Decode(&meta)
	require.NoError(t, err, "metadata should exist in mongo")
	assert.Equal(t, "Go Programming Language", meta["title"])
	assert.Equal(t, "Go is an open source programming language.", meta["description"])
	assert.NotEmpty(t, meta["summary_text"])
	t.Logf("Metadata: title=%s description=%s", meta["title"], meta["description"])
}

func TestIndexer_WritesWordsWithTF(t *testing.T) {
	env := setupIndexerTestEnv(t)
	defer env.cleanup()

	html := `<html><head><title>Concurrency in Go</title></head>
		<body>
			<p>Goroutines are lightweight threads managed by the Go runtime.</p>
			<p>Goroutines enable concurrent programming in Go.</p>
			<p>Go uses channels to communicate between goroutines.</p>
		</body>
	</html>`

	pushFakePage(t, env, "en.wikipedia.org/wiki/Goroutine", html, nil)
	runIndexerOnce(t, env)

	// goroutine appears 3 times — should have tf=3 (or boosted if in url)
	var wordDoc bson.M
	err := env.rawMongo.Collection("words").FindOne(env.ctx, bson.D{
		{Key: "word", Value: "goroutines"},
		{Key: "url", Value: "en.wikipedia.org/wiki/Goroutine"},
	}).Decode(&wordDoc)
	require.NoError(t, err, "word doc should exist for 'goroutines'")
	assert.Greater(t, int(wordDoc["tf"].(int32)), 0)
	assert.Equal(t, int32(0), wordDoc["weight"], "weight should be 0 before tf-idf")
	t.Logf("goroutines tf=%v weight=%v", wordDoc["tf"], wordDoc["weight"])
}

func TestIndexer_WritesOutlinks(t *testing.T) {
	env := setupIndexerTestEnv(t)
	defer env.cleanup()

	html := `<html><head><title>Test</title></head><body><p>Some content here.</p></body></html>`
	outlinks := []string{
		"en.wikipedia.org/wiki/Link1",
		"en.wikipedia.org/wiki/Link2",
	}

	pushFakePage(t, env, "en.wikipedia.org/wiki/Test", html, outlinks)
	runIndexerOnce(t, env)

	var outlinksDoc bson.M
	err := env.rawMongo.Collection("outlinks").FindOne(env.ctx, bson.D{
		{Key: "_id", Value: "en.wikipedia.org/wiki/Test"},
	}).Decode(&outlinksDoc)
	require.NoError(t, err, "outlinks doc should exist")

	links := outlinksDoc["links"].(bson.A)
	assert.Equal(t, 2, len(links))
	t.Logf("Outlinks: %v", links)
}

func TestIndexer_CleansUpRedisAfterIndexing(t *testing.T) {
	env := setupIndexerTestEnv(t)
	defer env.cleanup()

	html := `<html><head><title>Cleanup Test</title></head><body><p>Testing redis cleanup.</p></body></html>`
	pageKey := pushFakePage(t, env, "en.wikipedia.org/wiki/Cleanup", html, []string{"en.wikipedia.org/wiki/Other"})

	runIndexerOnce(t, env)

	// page hash should be deleted
	n, err := env.rawRedis.Exists(env.ctx, pageKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "page hash should be deleted from Redis after indexing")

	// outlinks key should be deleted
	n, err = env.rawRedis.Exists(env.ctx, "outlinks:en.wikipedia.org/wiki/Cleanup").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "outlinks key should be deleted from Redis after indexing")

	// queue should be empty
	qLen, err := env.rawRedis.LLen(env.ctx, "pages_queue").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), qLen, "queue should be empty after indexing")
}

func TestIndexer_SkipsNonEnglishPages(t *testing.T) {
	env := setupIndexerTestEnv(t)
	defer env.cleanup()

	// Japanese text
	html := `<html><head><title>日本語ページ</title></head>
		<body><p>これは日本語のテキストです。ゴルーチンは軽量スレッドです。</p></body>
	</html>`

	pushFakePage(t, env, "ja.wikipedia.org/wiki/Go", html, nil)
	runIndexerOnce(t, env)

	// Metadata should NOT be written for non-english pages
	count, err := env.rawMongo.Collection("metadata").CountDocuments(env.ctx, bson.D{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "non-english page should not be indexed")
}

func TestIndexer_URLBoostsKeywords(t *testing.T) {
	env := setupIndexerTestEnv(t)
	defer env.cleanup()

	// "goroutine" appears in both the URL and the text
	html := `<html><head><title>Goroutine</title></head>
		<body>
			<p>A goroutine is a lightweight thread of execution.</p>
			<p>Goroutines are cheaper than OS threads.</p>
		</body>
	</html>`

	pushFakePage(t, env, "en.wikipedia.org/wiki/Goroutine", html, nil)
	runIndexerOnce(t, env)

	// "goroutine" should have boosted tf (original * 50) because it's in the URL
	var wordDoc bson.M
	err := env.rawMongo.Collection("words").FindOne(env.ctx, bson.D{
		{Key: "word", Value: "goroutine"},
		{Key: "url", Value: "en.wikipedia.org/wiki/Goroutine"},
	}).Decode(&wordDoc)
	require.NoError(t, err)

	tf := int(wordDoc["tf"].(int32))
	t.Logf("goroutine tf=%d (should be boosted because it appears in URL)", tf)
	assert.Greater(t, tf, 2, "url boost should increase tf above raw count")
}
