package teacup

import (
	"context"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// Redis returns a redis client ready to use. The first call to Redis() will dial the Redis server
// and the provided context is used to control things like timeouts.
func (t *Teacup) Redis(ctx context.Context) (*redis.Client, error) {
	if t.redisClient != nil {
		return t.redisClient, nil
	}
	url, ok := os.LookupEnv("REDIS_URL")
	if ok {
		opt, err := redis.ParseURL(url)
		if err != nil {
			return nil, err
		}
		return redis.NewClient(opt), nil
	}
	pass, _ := t.Secret(ctx, "REDIS_PASSWORD")
	db, err := t.Option(ctx, "REDIS_DATABASE")
	database := 0
	if err == nil {
		database, err = strconv.Atoi(db)
		if err != nil {
			return nil, err
		}
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     t.ServiceAddr(ctx, "redis", 6379),
		Password: pass,
		DB:       database,
	})
	// TODO we should probably do a ping check or something...
	return rdb, nil
}
