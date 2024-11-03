// Package lru provides a ARC 2Q LRU-based cache wrapper client with expiration logic
package lru

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sinu5oid/cache"

	lru "github.com/hashicorp/golang-lru"
)

// Cache represents lru.ARCCache
//
// Items are subject of both eviction and TTL expiration
type Cache[T any] struct {
	storage    *lru.ARCCache
	rwQueue    *sync.Map
	defaultTTL *time.Duration
}

// NewCache creates a Cache instance with internal storages initialized and no TTL
func NewCache[T any](size int) (*Cache[T], error) {
	s, err := lru.NewARC(size)
	if err != nil {
		return nil, fmt.Errorf("could not create new LRU ARC cache: %w", err)
	}

	return &Cache[T]{
		storage:    s,
		rwQueue:    &sync.Map{},
		defaultTTL: nil,
	}, nil
}

// NewCacheWithTTL creates a Cache instance with internal storages initialized and TTL being set
func NewCacheWithTTL[T any](size int, defaultTTL time.Duration) (*Cache[T], error) {
	c, err := NewCache[T](size)
	if err != nil {
		return nil, err
	}

	return c.WithTTL(defaultTTL), nil
}

// WithTTL assigns provided ttl value
//
// Previous items are not updated automatically. Only newly added items would receive TTL settings
func (c *Cache[T]) WithTTL(ttl time.Duration) *Cache[T] {
	c.defaultTTL = &ttl
	return c
}

// Keys returns slice of stored keys
//
// The order of keys are not guaranteed
func (c *Cache[T]) Keys(_ context.Context) ([]string, error) {
	var keys []string
	for _, k := range c.storage.Keys() {
		keys = append(keys, k.(string))
	}
	return keys, nil
}

// Clear removes items from internal storages
func (c *Cache[T]) Clear() {
	c.storage.Purge()
}

// Get retrieves an item from cache by key. Does not return expired by TTL or otherwise evicted items
func (c *Cache[T]) Get(_ context.Context, key string) (T, error) {
	return c.get(key)
}

type getOrFetchResult[T any] struct {
	res T
	err error
}

// GetOrFetch tries to obtain cached value from internal storage. If multiple callers are accessing the same key,
// later callers join the wait queue until the result or error are received
//
// If the value was not found - calls provided fetcher function, saves received value to the cache.
func (c *Cache[T]) GetOrFetch(_ context.Context, key string, fetcher func() (T, error)) (T, error) {
	done := make(chan getOrFetchResult[T])
	close(done)

	lock, loaded := c.rwQueue.LoadOrStore(key, done)
	if loaded {
		c, ok := lock.(chan getOrFetchResult[T])
		if ok {
			res := <-c // wait here until other routine does the fetching
			return res.res, res.err
		}
	}

	result, err := c.get(key)
	if err == nil {
		return result, err
	}

	var missingEntryError cache.MissingEntryError
	if !errors.As(err, &missingEntryError) {
		return result, err
	}

	result, err = fetcher()
	done <- getOrFetchResult[T]{result, err}
	defer c.rwQueue.Delete(key)

	return result, err
}

// Set puts the provided value by cache key to internal storage
//
// By default uses TTL value provided during instantiation. If specific TTL is needed, use SetWithTTL
func (c *Cache[T]) Set(_ context.Context, key string, value T) error {
	c.set(key, value, nil)
	return nil
}

// GetMulti returns cached values by provided keys.
// Result slice may have fewer items than keys, it means that items by that key were not found
func (c *Cache[T]) GetMulti(_ context.Context, keys []string) ([]cache.StorageItemMulti[T], error) {
	res := make([]cache.StorageItemMulti[T], 0, len(keys))
	for _, key := range keys {
		val, err := c.get(key)
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
func (c *Cache[T]) SetMulti(_ context.Context, kvs []cache.StorageItemMulti[T]) error {
	for _, kv := range kvs {
		c.set(kv.Key, kv.Value, nil)
	}

	return nil
}

// Delete removes cached value from internal storage by key
func (c *Cache[T]) Delete(_ context.Context, key string) error {
	c.delete(key)
	return nil
}

// SetWithTTL puts provided value by cache key using provided ttl duration
func (c *Cache[T]) SetWithTTL(_ context.Context, key string, value T, ttl time.Duration) error {
	c.set(key, value, &ttl)
	return nil
}

// SetMultiWithTTL puts provided k/v pairs to cache using provided ttl duration
func (c *Cache[T]) SetMultiWithTTL(_ context.Context, kvs []cache.StorageItemMulti[T], ttl time.Duration) error {
	for _, kv := range kvs {
		c.set(kv.Key, kv.Value, &ttl)
	}

	return nil
}

type withTTL[T any] struct {
	UpdatedAt time.Time
	TTL       *time.Duration
	Value     T
}

func (c *Cache[T]) get(key string) (T, error) {
	value, ok := c.storage.Get(key)
	if !ok {
		return *new(T), cache.NewMissingEntryError(key)
	}

	casted, ok := value.(withTTL[T])
	if !ok {
		c.delete(key)

		return *new(T), cache.NewFailedToCastEntryError(key, nil)
	}

	if casted.TTL == nil {
		return casted.Value, nil
	}

	now := time.Now()
	if casted.UpdatedAt.Add(*casted.TTL).After(now) {
		return casted.Value, nil
	}

	c.delete(key)

	return *new(T), cache.NewMissingEntryError(key)
}

func (c *Cache[T]) set(key string, value T, ttl *time.Duration) {
	finalTTL := c.defaultTTL
	if ttl != nil {
		finalTTL = ttl
	}

	c.storage.Add(key, withTTL[T]{
		UpdatedAt: time.Now(),
		TTL:       finalTTL,
		Value:     value,
	})
}

func (c *Cache[T]) delete(key string) {
	c.storage.Remove(key)
}
