// Package cache provides a set of typed (using generics) cache wrappers
//
// Exact implementations following described Cacher, FetchingCacher
// and TTLCacher interfaces are available in subpackages
package cache

import (
	"context"
	"time"
)

// StorageItemMulti is a util struct describing key and value
//
// Actively used by Cacher.GetMulti and Cacher.SetMulti cache functions
type StorageItemMulti[T any] struct {
	Key   string
	Value T
}

type Cacher[T any] interface {
	Get(ctx context.Context, key string) (T, error)
	Set(ctx context.Context, key string, value T) error
	GetMulti(ctx context.Context, keys []string) ([]StorageItemMulti[T], error)
	SetMulti(ctx context.Context, kvs []StorageItemMulti[T]) error
	Delete(ctx context.Context, key string) error
}

type FetchingCacher[T any] interface {
	Cacher[T]
	GetOrFetch(ctx context.Context, key string, fetch func() (T, error)) (T, error)
}

type TTLCacher[T any] interface {
	Cacher[T]
	SetWithTTL(ctx context.Context, key string, value T, ttl time.Duration) error
	SetMultiWithTTL(ctx context.Context, kvs []StorageItemMulti[T], ttl time.Duration) error
}
