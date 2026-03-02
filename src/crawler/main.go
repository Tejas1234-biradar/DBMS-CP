package main

import (
	"flag"
	"log"
	"os"
	"sync"

	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/controllers"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/crawler"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/database"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/pages"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/utils"
)

func getEnv(key, fallback string) string {
	if value, exist := os.LookupEnv(key); exist {
		return value
	}
	return fallback
}
func main() {
	maxConcurrency := flag.Int("max-concurrency", 1, "Maximum number of concurrent workers")
	maxPages := flag.Int("max-pages", 100, "Maximum number of pages per batch")
	flag.Parse()
	//environment Variables
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnv("REDIS_DB", "0")
	startingURL := getEnv("STARTING_URL", "https://en.wikipedia.org/wiki/Agnes_Tachyon")
	//connect to redis
	db := &database.Database{}
	err := db.ConnectToRedis(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		log.Print("ErrorL %v\n", err)
		return
	}

	//Add start URL
	db.PushURL(startingURL, 0)
	log.Printf("Start: %v", startingURL)
	//Initiazize pageControllers
	pageController := controllers.NewPageController(db)
	linksController := controllers.NewLinksController(db)

	//Configure the crawler
	crawler := &crawler.CrawlerConfig{
		Mu:             &sync.Mutex{},
		Wg:             &sync.WaitGroup{},
		Pages:          make(map[string]*pages.Page),
		Outlinks:       make(map[string]*pages.PageNode),
		BackLinks:      make(map[string]*pages.PageNode),
		MaxPages:       *maxPages,
		MaxConcurrency: *maxConcurrency,
	}
	for {
		log.Printf("Checking the number of entries....\n")
		queueSize, err := db.GetIndexerQueueSize()
		if err != nil {
			log.Printf("Error getting the indexer queue: %v", err)
			return
		}
		if queueSize >= utils.MaxIndexerQueueSize {
			log.Printf("waiting for indexerQueue to get free...\n")
			for {
				sig, err := db.PopSignalQueue()
				if err != nil {
					log.Printf("Could not get the signal: %v", err)
					return
				}
				if sig == utils.ResumeCrawl {
					log.Printf("Resume Crawing \n")
					break
				}
			}
		}
		log.Printf("Spawning Workers....\n")
		for range crawler.MaxConcurrency {
			crawler.Wg.Add(1)
			go crawler.BFS(db)
		}
		crawler.Wg.Wait()
		pageController.SavePages(crawler)
		linksController.SaveLinks(crawler)
		crawler.Pages = make(map[string]*pages.Page)
		crawler.Outlinks = make(map[string]*pages.PageNode)
		crawler.BackLinks = make(map[string]*pages.PageNode)
	}
}
