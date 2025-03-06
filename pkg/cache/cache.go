package cache

import (
	"sync"
	"time"
)

// Item represents a cache item
type Item struct {
	Value      interface{}
	Expiration int64
}

// Cache represents a simple in-memory cache
type Cache struct {
	items             map[string]Item
	mu                sync.RWMutex
	defaultExpiration time.Duration
	cleanupInterval   time.Duration
	stopCleanup       chan bool
}

// New creates a new cache with the given TTL in seconds
func New(ttl int) (*Cache, error) {
	defaultExpiration := time.Duration(ttl) * time.Second
	cleanupInterval := time.Duration(ttl) * time.Second

	cache := &Cache{
		items:             make(map[string]Item),
		defaultExpiration: defaultExpiration,
		cleanupInterval:   cleanupInterval,
		stopCleanup:       make(chan bool),
	}

	// Start cleanup goroutine
	go cache.startCleanup()

	return cache, nil
}

// Set adds an item to the cache
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithExpiration(key, value, c.defaultExpiration)
}

// SetWithExpiration adds an item to the cache with a custom expiration
func (c *Cache) SetWithExpiration(key string, value interface{}, duration time.Duration) {
	var expiration int64

	if duration == 0 {
		// 0 means use default expiration
		duration = c.defaultExpiration
	}

	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = Item{
		Value:      value,
		Expiration: expiration,
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// Check if the item has expired
	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return nil, false
	}

	return item.Value, true
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]Item)
}

// Count returns the number of items in the cache
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Close stops the cleanup goroutine
func (c *Cache) Close() {
	c.stopCleanup <- true
}

// startCleanup starts the cleanup process
func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// deleteExpired deletes expired items
func (c *Cache) deleteExpired() {
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			delete(c.items, key)
		}
	}
}
