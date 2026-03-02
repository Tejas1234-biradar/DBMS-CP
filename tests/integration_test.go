//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pagesQueueKey  = "pages_queue"
	pageDataPrefix = "page_data"
	outlinksPrefix = "outlinks"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestLiveCrawlerPopulatesRedis(t *testing.T) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	defer rdb.Close()

	_, err := rdb.Ping(ctx).Result()
	require.NoError(t, err, "Redis must be running — start your crawler first")

	t.Logf("Waiting for crawler to push pages into %s...", pagesQueueKey)

	require.Eventually(t, func() bool {
		n, err := rdb.LLen(ctx, pagesQueueKey).Result()
		if err != nil {
			return false
		}
		t.Logf("Queue size: %d", n)
		return n >= 3
	}, 60*time.Second, 1*time.Second, "crawler should push pages to %s within 60s", pagesQueueKey)

	items, err := rdb.LRange(ctx, pagesQueueKey, 0, -1).Result()
	require.NoError(t, err)
	t.Logf("Pages in queue: %d", len(items))

	realPages := 0
	for _, pageKey := range items {
		// Skip control signals (items that aren't page_data: keys)
		if len(pageKey) < len(pageDataPrefix) || pageKey[:len(pageDataPrefix)] != pageDataPrefix {
			t.Logf("  skipping control signal: %q", pageKey)
			continue
		}

		fields, err := rdb.HGetAll(ctx, pageKey).Result()
		require.NoError(t, err)

		if len(fields) == 0 {
			t.Logf("  skipping empty key: %q", pageKey)
			continue
		}

		realPages++
		t.Logf("  key=%-60s url=%s", pageKey, fields["normalized_url"])

		assert.NotEmpty(t, fields["normalized_url"], "page %s missing normalized_url", pageKey)
		assert.NotEmpty(t, fields["html"], "page %s missing html", pageKey)
		assert.NotEmpty(t, fields["content_type"], "page %s missing content_type", pageKey)
		assert.NotEmpty(t, fields["last_crawled"], "page %s missing last_crawled", pageKey)
	}

	assert.Greater(t, realPages, 0, "should have at least one real page_data entry")

	// Verify outlinks exist for at least one page
	foundOutlinks := false
	for _, pageKey := range items {
		if len(pageKey) < len(pageDataPrefix) || pageKey[:len(pageDataPrefix)] != pageDataPrefix {
			continue
		}

		fields, _ := rdb.HGetAll(ctx, pageKey).Result()
		if url := fields["normalized_url"]; url != "" {
			count, _ := rdb.SCard(ctx, outlinksPrefix+":"+url).Result()
			if count > 0 {
				t.Logf("Outlinks for %s: %d", url, count)
				foundOutlinks = true
				break
			}
		}
	}
	assert.True(t, foundOutlinks, "at least one page should have outlinks in Redis")
}

