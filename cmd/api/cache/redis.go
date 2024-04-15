package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"strconv"
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

var client *Redis

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
	return client
}

func NewRedisClient() *Redis {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		Username: func() string {
			u := os.Getenv("REDIS_USERNAME")
			if u != "" {
				return u
			}
			return "default"
		}(),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB: func() int {
			d := os.Getenv("REDIS_DB")
			if i, err := strconv.Atoi(d); err == nil {
				return i
			}
			return 0
		}(),
	})
	return (*Redis)(client)
}

func (r *Redis) Get(key string) (*Item, error) {
	if !strings.HasPrefix(key, echo.MIMEApplicationJSON) {
		val, err := (*redis.Client)(r).Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("key %s not found %w", key, err)
		}
		if err != nil {
			return nil, err
		}
		var mimeType string
		if strings.Contains(key, ":") {
			mimeType = key[:strings.Index(key, ":")]
		}
		if strings.HasPrefix(mimeType, "http") {
			mimeType = MimeTypeFromURL(key)
		}
		if mimeType == "" {
			mimeType = echo.MIMEOctetStream
		}
		return &Item{
			LastAccess: time.Now().UTC(),
			MimeType:   mimeType,
			Blob:       val,
		}, nil
	}

	val, err := (*redis.Client)(r).JSONGet(ctx, key, "$").Result()
	if errors.Is(err, redis.Nil) || len(val) == 0 {
		return nil, fmt.Errorf("key %s not found %w", key, redis.Nil)
	}

	if err != nil {
		return nil, err
	}

	var items []any
	err = json.Unmarshal([]byte(val), &items)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("empty item %s %w", key, redis.Nil)
	}

	var item Item = Item{
		LastAccess: time.Now().UTC(),
		MimeType:   echo.MIMEApplicationJSON,
	}

	switch len(items) {
	case 0:
		return nil, fmt.Errorf("empty item %s %w", key, redis.Nil)
	case 1:
		item.Blob = []byte(val[1 : len(val)-1])
	default:
		item.Blob = []byte(val)
	}

	return &item, nil
}

type JSONItem struct {
	Blob       any       `json:"blob,omitempty"`
	LastAccess time.Time `json:"last_access"`
	MimeType   string    `json:"mime_type,omitempty"`
	HitCount   int       `json:"hit_count,omitempty"`
}

func (r *Redis) Set(key string, item *Item, duration time.Duration) error {
	if !strings.HasPrefix(key, item.MimeType) {
		key = fmt.Sprintf("%s:%s", item.MimeType, key)
	}

	if strings.HasSuffix(item.MimeType, "json") {
		cmd := (*redis.Client)(r).JSONSet(ctx, key, "$", item.Blob)
		if cmd.Err() != nil {
			return fmt.Errorf("failed to set item: %w", cmd.Err())
		}
		(*redis.Client)(r).ExpireAt(ctx, key, time.Now().UTC().Add(duration))

		return nil
	}

	cmd := (*redis.Client)(r).Set(ctx, key, item.Blob, duration)
	if cmd.Err() != nil {
		return fmt.Errorf("failed to set item: %w", cmd.Err())
	}

	return nil
}
