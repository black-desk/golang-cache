// Package cache provide a thread-safe KV-container. It has a significant high performance when a lot of goroutines read and write concurrently
package cache

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

// Errors
var ErrorKeyNotFound = errors.New("key not found!")
var ErrorSizeLimit = errors.New("Size limit")
var ErrorKeyExist = errors.New("Key exist")
var ErrorKeyHasBeenUpdate = errors.New("Key has been update")

// Cache is the KV-container
type Cache struct {
	expiration time.Duration             // expiration time for all key-value pair store in c container
	data       map[string]*item          // a map hold the pointer to item
	size       uint32                    // current amount of items in c Cache
	lock       sync.RWMutex              // a lock for the map,size,and heap above
	maxSize    uint32                    // the max amount of items Cache can contain
	onEvicted  func(string, interface{}) // a function to call when item evicted
	interval   time.Duration             // the interval for expiration check
	stop       chan int                  // the chan to close the timer in expiration check routine
}

// remove is a function called to stop the timer for expiration check
func remove(c *Cache) {
	c.stop <- 1
}

// watch is a goroutine which scan the key-value pairs in Cache then delete expired ones every interval
func (c *Cache) watch() {
	t := time.NewTicker(c.interval)
	for {
		select {
		case <-t.C:
			c.lock.Lock()
			now := time.Now()
			for k, p := range c.data {
				if p.updateTime.Add(c.expiration).Before(now) {
					delete(c.data, k)
					if c.onEvicted != nil {
						go c.onEvicted(k, *(p.objectPtr))
					}
				}
			}
			c.lock.Unlock()
		case <-c.stop:
			t.Stop()
			return
		}
	}
}

// NewCache get a new Cache, return the pointer point to it
func NewCache(expiration time.Duration, interval time.Duration, maxSize uint32, onEvicted func(string, interface{})) *Cache {
	c := &Cache{
		expiration: expiration,
		data:       map[string]*item{},
		lock:       sync.RWMutex{},
		size:       0,
		maxSize:    maxSize,
		onEvicted:  onEvicted,
		interval:   interval,
		stop:       make(chan int, 1),
	}
	go c.watch()
	runtime.SetFinalizer(c, remove)
	return c
}

// Get the value from key
func (c *Cache) Get(key string) (interface{}, error) {
	c.lock.RLock()
	itemPtr, found := c.data[key]
	if found {
		itemPtr.lock.RLock()
		c.lock.RUnlock()
	} else {
		c.lock.RUnlock()
		return nil, ErrorKeyNotFound
	}
	object := *itemPtr.objectPtr
	itemPtr.lock.RUnlock()
	return object, nil
}

// item hold a pointer to the actual object, the lock is for the ptr
type item struct {
	updateTime time.Time
	lock       sync.RWMutex
	objectPtr  *interface{}
}

// Set key-value pair into Cache, error when meet maxSize
func (c *Cache) Set(key string, value interface{}) error {
	valuePtr := &value
	return c.set(key, valuePtr, false)
}

// Add key-value pair into Cache, error when meet maxSize or the key exist
func (c *Cache) Add(key string, value interface{}) error {
	valuePtr := &value
	return c.set(key, valuePtr, true)
}

func (c *Cache) set(key string, valuePtr *interface{}, add bool) error {
	now := time.Now()
	c.lock.Lock()
	itemPtr, found := c.data[key]
	if !found {
		if c.size == c.maxSize {
			c.lock.Unlock()
			return ErrorSizeLimit
		}
		//create new item
		c.size++
		itemPtr = &item{}
		c.data[key] = itemPtr
		itemPtr.lock.Lock()
		c.lock.Unlock()
		itemPtr.updateTime = now
		itemPtr.objectPtr = valuePtr
		itemPtr.lock.Unlock()
		return nil
	} else if add {
		c.lock.Unlock()
		return ErrorKeyExist
	}
	itemPtr.lock.Lock()
	c.lock.Unlock()
	if itemPtr.updateTime.After(now) {
		itemPtr.lock.Unlock()
		return ErrorKeyHasBeenUpdate
	}
	itemPtr.updateTime = now
	itemPtr.objectPtr = valuePtr
	itemPtr.lock.Unlock()
	return nil
}
