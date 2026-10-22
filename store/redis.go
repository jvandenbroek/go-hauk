package store

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store interface {
	Get(ctx context.Context, key string, v any) error
	Set(ctx context.Context, key string, v any, expireAt time.Time) error
	SetTTL(ctx context.Context, key string, v any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	// ListPush appends v to the list at key, keeps only the last maxLen
	// entries and sets the list expiry. Only the new element is written.
	ListPush(ctx context.Context, key string, v any, maxLen int, expireAt time.Time) error
	// ListRange returns elements start..stop inclusive; negative indices count
	// from the tail (-1 = last), like Redis LRANGE. Missing key = empty.
	ListRange(ctx context.Context, key string, start, stop int) ([]json.RawMessage, error)
}

type Redis struct {
	client *redis.Client
	prefix string
}

func NewRedis(addr, password, prefix string) (*Redis, error) {
	var opts *redis.Options
	var err error

	// redis:// URL detection
	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") {
		opts, err = redis.ParseURL(addr)
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(addr, "/") {
		// unix socket
		opts = &redis.Options{
			Network:  "unix",
			Addr:     addr,
			Password: password,
		}
	} else {
		// host:port
		opts = &redis.Options{
			Addr:     addr,
			Password: password,
		}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Redis{
		client: client,
		prefix: prefix,
	}, nil
}

func (r *Redis) key(k string) string {
	return r.prefix + "-" + k
}

func (r *Redis) Get(ctx context.Context, key string, v any) error {
	data, err := r.client.Get(ctx, r.key(key)).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (r *Redis) Set(ctx context.Context, key string, v any, expireAt time.Time) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return r.setWithTTL(ctx, key, data, time.Until(expireAt))
}

func (r *Redis) SetTTL(ctx context.Context, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return r.setWithTTL(ctx, key, data, ttl)
}

// setWithTTL refuses to write keys without a positive TTL. go-redis sends a
// plain SET (no expiry) for ttl <= 0, which would leave already-expired
// sessions/shares in redis forever.
func (r *Redis) setWithTTL(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return r.client.Del(ctx, r.key(key)).Err()
	}
	return r.client.Set(ctx, r.key(key), data, ttl).Err()
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.key(key)).Err()
}

func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, r.key(key)).Result()
	return n > 0, err
}

func (r *Redis) ListPush(ctx context.Context, key string, v any, maxLen int, expireAt time.Time) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	k := r.key(key)
	_, err = r.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.RPush(ctx, k, data)
		if maxLen > 0 {
			p.LTrim(ctx, k, int64(-maxLen), -1)
		}
		p.ExpireAt(ctx, k, expireAt)
		return nil
	})
	return err
}

func (r *Redis) ListRange(ctx context.Context, key string, start, stop int) ([]json.RawMessage, error) {
	vals, err := r.client.LRange(ctx, r.key(key), int64(start), int64(stop)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, len(vals))
	for i, v := range vals {
		out[i] = json.RawMessage(v)
	}
	return out, nil
}

func (r *Redis) Close() error {
	return r.client.Close()
}
