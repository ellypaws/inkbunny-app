package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func Init() {
	client = NewRedisClient()

	_, err := client.Ping(ctx).Result()
	if err == nil {
		Initialized = true
	}

	if addr := os.Getenv("REDIS_HOST"); !Initialized {
		log.Printf("warning: redis %s not initialized", addr)
	} else {
		log.Printf("redis initialized: %v", addr)
	}
}

type Redis redis.Client

var (
	client      *Redis
	ctx         = context.Background()
	Initialized bool
)

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
	options := &redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Username: "default",
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	}
	if u := os.Getenv("REDIS_USERNAME"); u != "" {
		options.Username = u
	}
	if d := os.Getenv("REDIS_DB"); d != "" {
		if i, err := strconv.Atoi(d); err == nil {
			options.DB = i
		}
	}
	return (*Redis)(redis.NewClient(options))
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
		if p := strings.Index(key, ":"); p != -1 {
			mimeType = key[:p]
		}
		if strings.HasPrefix(mimeType, "http") {
			m := MimeTypeFromURL(key)
			if m != "" && mimeType != m {
				log.Printf("warning: mime type mismatch %s %s", mimeType, m)
				mimeType = m
			}
		}
		if mimeType == "" {
			log.Printf("warning: mime type not set for %s", key)
			mimeType = echo.MIMEOctetStream
		}
		return &Item{
			MimeType:   mimeType,
			Blob:       val,
			lastAccess: time.Now().UTC(),
		}, nil
	}

	val, err := (*redis.Client)(r).JSONGet(ctx, key, "$").Result()
	if errors.Is(err, redis.Nil) || len(val) == 0 {
		return nil, fmt.Errorf("key %s not found %w", key, redis.Nil)
	}

	if err != nil {
		return nil, err
	}

	var items []json.RawMessage
	err = json.Unmarshal([]byte(val), &items)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("empty item %s %w", key, redis.Nil)
	}

	var item Item = Item{
		MimeType:   echo.MIMEApplicationJSON,
		lastAccess: time.Now().UTC(),
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

func (r *Redis) MGet(keys ...string) (map[string]*Item, error) {
	items := make(map[string]*Item)
	var foundAny bool

	for _, key := range keys {
		if !strings.HasPrefix(key, echo.MIMEApplicationJSON) {
			val, err := (*redis.Client)(r).Get(ctx, key).Bytes()
			if errors.Is(err, redis.Nil) {
				items[key] = nil
				continue
			}
			if err != nil {
				return nil, err
			}
			foundAny = true
			var mimeType string
			if p := strings.Index(key, ":"); p != -1 {
				mimeType = key[:p]
			}
			if strings.HasPrefix(mimeType, "http") {
				m := MimeTypeFromURL(key)
				if m != "" && mimeType != m {
					log.Printf("warning: mime type mismatch %s %s", mimeType, m)
					mimeType = m
				}
			}
			if mimeType == "" {
				log.Printf("warning: mime type not set for %s", key)
				mimeType = echo.MIMEOctetStream
			}

			items[key] = &Item{
				MimeType:   mimeType,
				Blob:       val,
				lastAccess: time.Now().UTC(),
			}

			continue
		}

		val, err := (*redis.Client)(r).JSONGet(ctx, key, "$").Result()
		if errors.Is(err, redis.Nil) || len(val) == 0 {
			items[key] = nil
			continue
		}
		if err != nil {
			return nil, err
		}

		if len(val) == 0 {
			items[key] = nil
			continue
		}

		var item Item = Item{
			MimeType:   echo.MIMEApplicationJSON,
			lastAccess: time.Now().UTC(),
		}

		var values []json.RawMessage
		err = json.Unmarshal([]byte(val), &values)
		if err != nil {
			return nil, err
		}

		switch len(values) {
		case 0:
			items[key] = nil
			continue
		case 1:
			item.Blob = []byte(val[1 : len(val)-1])
		default:
			item.Blob = []byte(val)
		}

		foundAny = true
		items[key] = &item
	}

	if !foundAny {
		return nil, fmt.Errorf("all keys not found %w", redis.Nil)
	}

	return items, nil
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
		if duration > 0 {
			(*redis.Client)(r).ExpireAt(ctx, key, time.Now().UTC().Add(duration))
		}

		return nil
	}

	cmd := (*redis.Client)(r).Set(ctx, key, item.Blob, duration)
	if cmd.Err() != nil {
		return fmt.Errorf("failed to set item: %w", cmd.Err())
	}

	return nil
}
