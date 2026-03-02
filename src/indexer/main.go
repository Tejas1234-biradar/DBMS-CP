package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/data"
	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/schemas"
	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	WORDS_OP_THRESHOLD    = 1000
	METADATA_OP_THRESHOLD = 100
	OUTLINKS_OP_THRESHOLD = 100
)

var (
	wordsOps    []mongo.WriteModel
	metadataOps []mongo.WriteModel
	outlinksOps []mongo.WriteModel
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// REDIS ENV VARIABLES
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	// MONGO ENV VARIABLES
	mongoHost := getEnv("MONGO_HOST", "localhost")
	mongoPort, _ := strconv.Atoi(getEnv("MONGO_PORT", "27017"))
	mongoPassword := getEnv("MONGO_PASSWORD", "")
	mongoDB := getEnv("MONGO_DB", "test")
	mongoUsername := getEnv("MONGO_USERNAME", "")

	// CONNECT TO REDIS
	log.Println("Initializing Redis...")
	redisClient := data.NewRedisClient(redisHost+":"+redisPort, redisPassword, redisDB)
	if redisClient == nil {
		log.Fatal("Could not initialize Redis, exiting...")
	}

	// CONNECT TO MONGO
	log.Println("Initializing Mongo...")
	mongoClient, err := data.NewMongoClient(mongoHost, mongoUsername, mongoPassword, mongoDB, mongoPort)
	if err != nil {
		log.Fatalf("Could not initialize Mongo: %v, exiting...", err)
	}

	// performBulkOperations equiv of perform_bulk_operations()
	performBulkOperations := func() {
		if len(wordsOps) >= WORDS_OP_THRESHOLD {
			log.Println("Performing words bulk operations...")
			mongoClient.CreateWordsBulk(ctx, wordsOps)
			wordsOps = nil
		}
		if len(metadataOps) >= METADATA_OP_THRESHOLD {
			log.Println("Performing metadata bulk operations...")
			mongoClient.CreateMetadataBulk(ctx, metadataOps)
			metadataOps = nil
		}
		if len(outlinksOps) >= OUTLINKS_OP_THRESHOLD {
			log.Println("Performing outlinks bulk operations...")
			mongoClient.CreateOutlinksBulk(ctx, outlinksOps)
			outlinksOps = nil
		}
	}

	// flushAll — equiv of handle_exit final bulk operations
	flushAll := func() {
		log.Println("Performing final bulk operations...")
		mongoClient.CreateWordsBulk(ctx, wordsOps)
		mongoClient.CreateMetadataBulk(ctx, metadataOps)
		mongoClient.CreateOutlinksBulk(ctx, outlinksOps)
	}

	// Handle shutdown signal in background
	go func() {
		<-sigChan
		log.Println("Termination signal received - shutting down...")
		flushAll()
		cancel()
		os.Exit(0)
	}()

	// INDEXING LOOP
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		queueSize, err := redisClient.GetQueueSize()
		if err != nil {
			log.Printf("Error getting queue size: %v", err)
			continue
		}
		if queueSize == 0 {
			log.Println("Queue empty, signaling crawler...")
			redisClient.SignalCrawler()
		}

		log.Println("Waiting for message queue...")

		pageID, err := redisClient.PopPage()
		if err != nil || pageID == "" {
			log.Printf("Could not fetch data from indexer queue: %v", err)
			continue
		}
		if pageID == "RESUME_CRAWL" || pageID == "dummy_page" {
			log.Printf("Skipping control signal: %s", pageID)
			continue
		}

		log.Printf("Fetching %s...", pageID)
		pageData, err := redisClient.GetPageData(pageID)
		if err != nil || len(pageData) == 0 {
			log.Printf("Could not fetch %s, skipping...", pageID)
			continue
		}

		normalizedURL := pageData["normalized_url"]
		html := pageData["html"]
		log.Printf("Page url: %s", normalizedURL)
		preview := html
		if len(preview) > 15 {
			preview = preview[:15] + "..."
		}
		log.Printf("Page html: %s", preview)

		log.Printf("Getting %s metadata...", pageID)
		oldMetadata, err := mongoClient.GetMetadata(ctx, normalizedURL)
		if err == nil && oldMetadata != nil && oldMetadata.LastCrawled.String() == pageData["last_crawled"] {
			log.Printf("No updates to %s, skipping...", normalizedURL)
			continue
		}

		log.Printf("Parsing html data for %s...", pageID)
		htmlData, err := utils.GetHTMLData(html)
		if err != nil || htmlData == nil {
			log.Printf("Could not parse html data for %s, skipping...", pageID)
			continue
		}

		if htmlData.Language != "en" {
			log.Printf("%s not english, skipping...", pageID)
			continue
		}

		if len(htmlData.Text) == 0 {
			log.Printf("Could not process text for %s, skipping...", pageID)
			continue
		}

		// equiv of words_frequency = Counter(text)
		log.Printf("Counting words from %s...", pageID)
		wordFreq := make(map[string]int)
		for _, word := range htmlData.Text {
			wordFreq[word]++
		}

		// equiv of keywords = dict(words_frequency.most_common(MAX_INDEX_WORDS))
		keywords := topN(wordFreq, utils.MAX_INDEX_WORDS)

		// equiv of words_in_url boost
		log.Printf("Checking words in url %s...", normalizedURL)
		wordsInURL := utils.SplitURL(normalizedURL)
		for _, word := range wordsInURL {
			if past, ok := keywords[word]; ok && past != 0 {
				keywords[word] = past * 50
			} else {
				keywords[word] = 10
			}
		}

		// equiv of create_words_entry_operation loop
		for word, frequency := range keywords {
			op := mongoClient.CreateWordsEntryOperation(word, normalizedURL, frequency)
			wordsOps = append(wordsOps, op)
		}

		// equiv of create_metadata_entry_operation
		page := &schemas.Page{
			NormalizedURL: normalizedURL,
			LastCrawled:   parseTime(pageData["last_crawled"]),
		}
		metaSchema := &schemas.Metadata{
			Title:       htmlData.Title,
			Description: htmlData.Description,
			SummaryText: htmlData.SummaryText,
		}
		metaOp := mongoClient.CreateMetadataEntryOperation(page, metaSchema, keywords)
		metadataOps = append(metadataOps, metaOp)

		// equiv of outlinks = redis.get_outlinks(normalized_url)
		outlinks, err := redisClient.GetOutlinks(normalizedURL)
		if err == nil && len(outlinks) > 0 {
			outlinksObj := &schemas.Outlinks{
				ID:    normalizedURL,
				Links: sliceToSet(outlinks),
			}
			outlinksOp := mongoClient.CreateOutlinksEntryOperation(outlinksObj)
			outlinksOps = append(outlinksOps, outlinksOp)
		}

		// equiv of wordsSet = {word.lower() for word in text}
		wordsSet := make([]string, 0, len(htmlData.Text))
		seen := make(map[string]struct{})
		for _, word := range htmlData.Text {
			if _, ok := seen[word]; !ok {
				wordsSet = append(wordsSet, word)
				seen[word] = struct{}{}
			}
		}
		mongoClient.AddWordsToDictionary(ctx, wordsSet)
		log.Println("Added words to dictionary...")

		log.Println("Deleting page data from Redis...")
		redisClient.DeletePageData(pageID)
		redisClient.DeleteOutlinks(normalizedURL)

		performBulkOperations()
	}
}

// topN equiv of Counter.most_common(n)
func topN(freq map[string]int, n int) map[string]int {
	type kv struct {
		key string
		val int
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].val > sorted[j].val
	})
	result := make(map[string]int)
	for i, kv := range sorted {
		if i >= n {
			break
		}
		result[kv.key] = kv.val
	}
	return result
}

func sliceToSet(links []string) map[string]struct{} {
	set := make(map[string]struct{}, len(links))
	for _, l := range links {
		set[l] = struct{}{}
	}
	return set
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC1123, s)
	return t
}
