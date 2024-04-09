package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/bytes"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"strings"
	"time"
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
	c.Logger().Debugf("Retrieving %s", key)
	val, err := (*redis.Client)(r).JSONGet(ctx, key, "$").Result()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("key %s not found", key)
	}
	if err != nil {
		return nil, err
	}
	var items []JSONItem
	err = json.Unmarshal([]byte(val), &items)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("empty item %s", key)
	}

	var item Item = Item{
		LastAccess: time.Now().UTC(),
		MimeType:   items[0].MimeType,
		HitCount:   items[0].HitCount + 1,
	}

	if item.MimeType == echo.MIMEApplicationJSON {
		b, err := json.Marshal(items[0].Blob)
		if err != nil {
			return nil, err
		}
		item.Blob = b
	}

	c.Logger().Infof("Cache hit for %s", key)
	return &item, nil
}

type JSONItem struct {
	Blob       any       `json:"blob,omitempty"`
	LastAccess time.Time `json:"last_access"`
	MimeType   string    `json:"mime_type,omitempty"`
	HitCount   int       `json:"hit_count,omitempty"`
}

func (r *Redis) Set(c echo.Context, key string, item *Item) error {
	if strings.HasPrefix(item.MimeType, "image") {
		// set as binary
	}
	i, err := item.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}
	cmd := (*redis.Client)(r).JSONSet(ctx, key, "$", i)
	if cmd.Err() != nil {
		return fmt.Errorf("failed to set item: %w", cmd.Err())
	}
	c.Logger().Debugf("Cached %s %s %dKiB", key, item.MimeType, len(i)/bytes.KiB)
	return nil
}
