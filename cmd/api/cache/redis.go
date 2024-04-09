package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
)

func init() {
	if e := os.Getenv("REDIS_HOST"); e != "" {
		addr = e
	}

	client = NewRedisClient()

	_, err := client.Ping(ctx).Result()
	if err == nil {
		Initialized = true
	}

	if !Initialized {
		log.Printf("warning: redis not initialized")
	} else {
		log.Printf("redis initialized: %v", addr)
	}
}

type Redis redis.Client

var client *redis.Client

var addr string = "localhost:6379"

var ctx = context.Background()

var Initialized bool

func Context() context.Context {
	return ctx
}

func NewContext() context.Context {
	ctx = context.Background()
	return ctx
}

func RedisClient() *Redis {
	return (*Redis)(client)
}

func NewRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: "default",
		Password: "",
		DB:       0,
	})
	return client
}

func (r *Redis) Get(c echo.Context, key string) (*Item, error) {
	c.Logger().Debugf("retrieving %s from redis", key)
	val, err := (*redis.Client)(r).Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("key %s not found", key)
	}
	if err != nil {
		return nil, err
	}
	var item Item
	err = item.UnmarshalBinary([]byte(val))
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Redis) Set(c echo.Context, key string, item *Item) error {
	bin, err := item.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}
	cmd := (*redis.Client)(r).Set(ctx, key, bin, 0)
	if cmd.Err() != nil {
		return fmt.Errorf("failed to set item: %w", cmd.Err())
	}
	c.Logger().Debugf("[redis] cached %s %s %dKiB", key, item.MimeType, len(item.Blob)/bytes.KiB)
	return nil
}
