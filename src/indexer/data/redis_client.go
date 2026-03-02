package data

import (
	"context"
	"log"

	"github.com/Tejas1234-biradar/DBMS-CP/src/indexer/utils"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("failed to connect to Redis:%v", err)
	}
	log.Println("Succesfully connected to Redis!")
	return &RedisClient{
		Client: rdb,
		Ctx:    ctx,
	}
}

// Redis Data
func (r *RedisClient) PopPage() (string, error) {
	result, err := r.Client.RPop(r.Ctx, utils.INDEXER_QUEUE_KEY).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}
func (r *RedisClient) GetQueueSize() (int64, error) {
	return r.Client.LLen(r.Ctx, utils.INDEXER_QUEUE_KEY).Result()
}
func (r *RedisClient) SignalCrawler() error {
	return r.Client.LPush(r.Ctx, utils.SIGNAL_QUEUE_KEY, utils.RESUME_CRAWL).Err()
}

// page data
func (r *RedisClient) GetPageData(key string) (map[string]string, error) {
	return r.Client.HGetAll(r.Ctx, key).Result()
}
func (r *RedisClient) DeletePageData(key string) error {
	return r.Client.Del(r.Ctx, key).Err()
}

// Outlinks
func (r *RedisClient) GetOutlinks(normalizedURL string) ([]string, error) {
	key := utils.OUTLINKS_PREFIX + ":" + normalizedURL
	return r.Client.SMembers(r.Ctx, key).Result()
}
func (r *RedisClient) DeleteOutlinks(normalizedURL string) error {
	key := utils.OUTLINKS_PREFIX + ":" + normalizedURL
	return r.Client.Del(r.Ctx, key).Err()
}
