package handlers

import (
    "strings"
    "sync"
    "time"
)

type cacheItem struct {
    payload []byte
    expires time.Time
}

type responseCache struct {
    mu    sync.RWMutex
    items map[string]cacheItem
    ttl   time.Duration
}

func newResponseCache(ttl time.Duration) *responseCache {
    return &responseCache{
        items: make(map[string]cacheItem),
        ttl:   ttl,
    }
}

func (c *responseCache) getOrSet(key string, build func() ([]byte, error)) ([]byte, error) {
    now := time.Now()

    c.mu.RLock()
    item, ok := c.items[key]
    c.mu.RUnlock()

    if ok && now.Before(item.expires) {
        return item.payload, nil
    }

    payload, err := build()
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.items[key] = cacheItem{
        payload: payload,
        expires: now.Add(c.ttl),
    }
    c.mu.Unlock()

    return payload, nil
}

func (c *responseCache) clear() {
    c.mu.Lock()
    c.items = make(map[string]cacheItem)
    c.mu.Unlock()
}

func (c *responseCache) clearPrefix(prefix string) {
    c.mu.Lock()
    for key := range c.items {
        if strings.HasPrefix(key, prefix) {
            delete(c.items, key)
        }
    }
    c.mu.Unlock()
}
