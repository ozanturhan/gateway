package gateway

import (
	"sync"
	"time"
)

type QueryCache struct {
	Query    *string
	LastUsed time.Time
}

type InMemoryCache struct {
	cache map[string]*QueryCache
	ttl   time.Duration
	// the automatic query plan cache needs to clear itself of query plans that have been used
	// recently. This coordination requires a channel over which events can be trigger whenever
	// a query is fired, triggering a check to clean up other queries.
	retrievedPlan chan bool
	// a boolean to track if there is a timer that needs to be reset
	resetTimer bool
	// a mutex on the timer bool
	timeMutex sync.Mutex
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		cache: map[string]*QueryCache{},
		// default cache lifetime of 3 days
		ttl:           10 * 24 * time.Hour,
		retrievedPlan: make(chan bool),
		resetTimer:    false,
	}
}

// WithCacheTTL updates and returns the cache with the new cache lifetime. Queries that haven't been
// used in that long are cleaned up on the next query.
func (c *InMemoryCache) WithCacheTTL(duration time.Duration) Cache {
	return &InMemoryCache{
		cache:         map[string]*QueryCache{},
		ttl:           duration,
		retrievedPlan: make(chan bool),
		resetTimer:    false,
	}
}

func (c *InMemoryCache) Set(key *string, value *string) {
	// when we're done with retrieving the value we have to clear the cache
	defer func() {
		// spawn a goroutine that might be responsible for clearing the cache
		go func() {
			// check if there is a timer to reset
			c.timeMutex.Lock()
			resetTimer := c.resetTimer
			c.timeMutex.Unlock()

			// if there is already a goroutine that's waiting to clean things up
			if resetTimer {
				// just reset their time
				c.retrievedPlan <- true
				// and we're done
				return
			}
			c.timeMutex.Lock()
			c.resetTimer = true
			c.timeMutex.Unlock()

			// otherwise this is the goroutine responsible for cleaning up the cache
			timer := time.NewTimer(c.ttl)

			// we will have to consume more than one input
		TRUE_LOOP:
			for {
				select {
				// if another plan was retrieved
				case <-c.retrievedPlan:
					// reset the time
					timer.Reset(c.ttl)

				// if the timer dinged
				case <-timer.C:
					// there is no longer a timer to reset
					c.timeMutex.Lock()
					c.resetTimer = false
					c.timeMutex.Unlock()

					// loop over every time in the cache
					for key, cacheItem := range c.cache {
						// if the cached query hasn't been used recently enough
						if cacheItem.LastUsed.Before(time.Now().Add(-c.ttl)) {
							// delete it from the cache
							delete(c.cache, key)
						}
					}

					// stop consuming
					break TRUE_LOOP
				}
			}

		}()
	}()

	c.cache[*key] = &QueryCache{
		Query:    value,
		LastUsed: time.Now(),
	}
}

func (c *InMemoryCache) Get(key *string) (*string, bool) {
	if cached, hasCachedValue := c.cache[*key]; hasCachedValue {
		// update the last used
		// cached.LastUsed = time.Now()
		// return it
		return cached.Query, true
	}

	return nil, false
}
