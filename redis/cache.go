// Package redis provides a typed redis cache wrapper
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sinu5oid/cache"

	rc "github.com/go-redis/cache/v9"
)

// Cache represents typed go-redis/cache wrapped
type Cache[T any] struct {
	storage *rc.Cache
	baseKey string
}

// NewCache creates a Cache instance with internal storages initialized and no TTL
func NewCache[T any](cache *rc.Cache, baseKey string) (*Cache[T], error) {
	return &Cache[T]{
		storage: cache,
		baseKey: baseKey,
	}, nil
}

// Get retrieves an item from cache by key. Does not return expired by TTL items
func (c *Cache[T]) Get(ctx context.Context, key string) (T, error) {
	return c.get(ctx, key, nil)
}

// GetOrFetch tries to obtain cached value from internal storage. If multiple callers are accessing the same key,
// later callers join the wait queue until the result or error are received
//
// If the value was not found - calls provided fetcher function, saves received value to the cache.
func (c *Cache[T]) GetOrFetch(ctx context.Context, key string, f func() (T, error)) (T, error) {
	return c.get(ctx, key, f)
}

// Set puts the provided value by cache key
//
// By default uses no TTL
func (c *Cache[T]) Set(ctx context.Context, key string, value T) error {
	return c.set(ctx, key, value, nil)
}

// GetMulti returns cached values by provided keys.
// Result slice may have fewer items than keys, it means that items by that key were not found
func (c *Cache[T]) GetMulti(ctx context.Context, keys []string) ([]cache.StorageItemMulti[T], error) {
	res := make([]cache.StorageItemMulti[T], 0, len(keys))
	for _, key := range keys {
		val, err := c.get(ctx, key, nil)
		if err != nil {
			continue
		}

		item := cache.StorageItemMulti[T]{
			Key:   key,
			Value: val,
		}
		res = append(res, item)
	}

	return res, nil
}

// SetMulti puts provided k/v pairs to cache
func (c *Cache[T]) SetMulti(ctx context.Context, kvs []cache.StorageItemMulti[T]) error {
	errs := make([]error, 0, len(kvs))
	for _, kv := range kvs {
		errs = append(errs, c.set(ctx, kv.Key, kv.Value, nil))
	}

	return errors.Join(errs...)
}

// Delete removes cached value by key
func (c *Cache[T]) Delete(ctx context.Context, key string) error {
	return c.delete(ctx, key)
}

// SetWithTTL puts provided value by cache key using provided ttl duration
func (c *Cache[T]) SetWithTTL(ctx context.Context, key string, value T, ttl time.Duration) error {
	return c.set(ctx, key, value, &ttl)
}

// SetMultiWithTTL puts provided k/v pairs to cache using provided ttl duration
func (c *Cache[T]) SetMultiWithTTL(ctx context.Context, kvs []cache.StorageItemMulti[T], ttl time.Duration) error {
	errs := make([]error, 0, len(kvs))
	for _, kv := range kvs {
		errs = append(errs, c.set(ctx, kv.Key, kv.Value, &ttl))
	}

	return errors.Join(errs...)
}

func (c *Cache[T]) get(ctx context.Context, key string, do func() (T, error)) (T, error) {
	out := new(T)

	item := rc.Item{
		Ctx:   ctx,
		Key:   c.formatKey(key),
		Value: &out,
	}

	if do != nil {
		item.Do = func(_ *rc.Item) (interface{}, error) {
			return do()
		}
	}

	err := c.storage.Once(&item)
	if err != nil {
		if errors.Is(err, rc.ErrCacheMiss) {
			return *out, cache.NewMissingEntryError(key)
		}

		return *new(T), fmt.Errorf("failed to get value from redis cache: %w", err)
	}

	return *out, nil
}

func (c *Cache[T]) set(ctx context.Context, key string, value T, ttl *time.Duration) error {
	item := &rc.Item{
		Ctx:   ctx,
		Key:   c.formatKey(key),
		Value: value,
	}

	if ttl != nil {
		item.TTL = *ttl
	}

	return c.storage.Set(item)
}

func (c *Cache[T]) delete(ctx context.Context, key string) error {
	return c.storage.Delete(ctx, key)
}

func (c *Cache[T]) formatKey(key string) string {
	return fmt.Sprintf("%s:%s", c.baseKey, key)
}
