package controllers

import (
	"fmt"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/crawler"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/database"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/pages"
	"github.com/Tejas1234-biradar/DBMS-CP/src/crawler/core/utils"
	"github.com/redis/go-redis/v9"
	"log"
)

// made a new directory to avoid circular dependency
// used to write in the indexer database
type PageController struct {
	db *database.Database
}

func NewPageController(db *database.Database) *PageController {
	return &PageController{
		db: db,
	}
}
func (pgc *PageController) GetAllPages() map[string]*pages.Page {
	log.Printf("fetching data from redis..\n")
	redisPages := make(map[string]*pages.Page)
	keys, err := pgc.db.Client.Keys(pgc.db.Context, utils.PagePrefix+":").Result()
	if err != nil {
		log.Printf("Error fetching Data from Redis: %v\n", err)
		return nil
	}
	// Pipelining is a technique to extremely speed up processing by packing
	// operations to batches, send them at once to Redis and read a replies in a
	// single step.
	// See https://redis.io/topics/pipelining
	pipeline := pgc.db.Client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipeline.HGetAll(pgc.db.Context, key)
	}
	_, err = pipeline.Exec(pgc.db.Context)
	if err != nil {
		log.Printf("Error fetching data from redis pipeline: %v", err)
		return nil
	}
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			log.Printf("Error fetching pipeline result: %v", err)
			return nil
		}
		page, err := pages.DehashPage(data)
		if err != nil {
			fmt.Printf("Error dehashing the page: %v", err)
		}
		redisPages[page.NormalizedURL] = page
	}
	return redisPages
}
func (pgc *PageController) SavePages(c *crawler.CrawlerConfig) {
	data := c.Pages
	log.Printf("writing %d entries to the db....\n", len(data))

	pipeline := pgc.db.Client.Pipeline()
	for _, page := range data {
		pageHash, err := pages.HashPage(page)
		if err != nil {
			log.Printf("Error hashing page %s: %v", page.NormalizedURL, err)
			continue
		}

		pageKey := utils.PagePrefix + ":" + page.NormalizedURL

		// Write the page data to its OWN key
		pipeline.HSet(pgc.db.Context, pageKey, pageHash)

		// Push that key onto the indexer queue list
		pipeline.LPush(pgc.db.Context, utils.IndexerQueueKey, pageKey)
	}

	results, err := pipeline.Exec(pgc.db.Context)
	if err != nil {
		log.Printf("error executing pipeline: %v", err)
	} else {
		log.Printf("successfully wrote %d entries, %d pipeline results", len(data), len(results))
		// Log any individual command errors
		for i, result := range results {
			if result.Err() != nil {
				log.Printf("  pipeline cmd %d failed: %v", i, result.Err())
			}
		}
	}
}
