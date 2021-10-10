package gateway

import (
	"github.com/bradfitz/gomemcache/memcache"
	"time"
)

type MemcachedCache struct {
	cache *memcache.Client
	ttl   time.Duration
}

func NewMemcachedCache(server ...string) *MemcachedCache {
	mc := memcache.New(server...)
	return &MemcachedCache{
		ttl:   10 * 24 * time.Hour,
		cache: mc,
	}
}

// WithCacheTTL updates and returns the cache with the new cache lifetime. Queries that haven't been
// used in that long are cleaned up on the next query.
func (c *MemcachedCache) WithCacheTTL(duration time.Duration) Cache {
	return &MemcachedCache{
		ttl: duration,
	}
}

func (c *MemcachedCache) Set(key *string, value *string) {
	err := c.cache.Set(&memcache.Item{Key: *key, Value: []byte(*value), Expiration: int32(int(c.ttl.Seconds()))})
	if err != nil {
		panic(err)
	}
}

func (c *MemcachedCache) Get(key *string) (*string, bool) {
	planData, err := c.cache.Get(*key)

	if err != nil {
		return nil, false
	}

	if planData != nil {
		cached := string(planData.Value)

		return &cached, true
	}

	return nil, false
}
