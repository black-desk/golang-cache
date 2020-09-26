// Package cache provide a thread-safe KV-container
package cache

import (
	"errors"
	"sync"
	"time"
)

// Cache is a KV-container
type Cache struct {
	expiration time.Duration             // expiration time for all key-value pair store in this container
	data       map[string]*item          // a map hold the pointer to item
	size       uint32                    // current amount of items in this Cache
	lock       sync.RWMutex              // a lock for the map,size,and heap above
	maxSize    uint32                    // the max amount of items Cache can contain
	onEvicted  func(string, interface{}) // a function to call when item evicted
	interval   time.Duration
}

// NewCache get a new Cache, return the pointer to it
func NewCache(expiration time.Duration, interval time.Duration, maxSize uint32, onEvicted func(string, interface{})) *Cache {
	c := &Cache{
		expiration: expiration,
		data:       map[string]*item{},
		lock:       sync.RWMutex{},
		size:       0,
		maxSize:    maxSize,
		onEvicted:  onEvicted,
		interval:   interval,
	}
	go c.watch()
	return c
}

// Get the value bind to the key
func (this *Cache) Get(key string) (interface{}, error) {
	this.lock.RLock()
	itemPtr, found := this.data[key]
	if found {
		itemPtr.lock.RLock()
		this.lock.RUnlock()
	} else {
		this.lock.RUnlock()
		return nil, errors.New(key + " not found!")
	}
	object := *itemPtr.objectPtr
	itemPtr.lock.RUnlock()
	return object, nil
}

// item hold a pointer to the actual object, the lock is for the ptr, cnt count the pointer point to this item in queue
type item struct {
	updateTime time.Time
	expiration int64
	lock       sync.RWMutex
	objectPtr  *interface{}
}

// Set bind map key to value, error if set fail, probably because meet the maxSize
func (this *Cache) Set(key string, value interface{}) error {
	valuePtr := &value
	return this.set(key, valuePtr, false)
}

// Add bind map key to value,only when the key not exist, error if set fail, probably because meet the maxSize or the key exist
func (this *Cache) Add(key string, value interface{}) error {
	valuePtr := &value
	return this.set(key, valuePtr, true)
}

// set bind map the key to value, add means only set the key when key not exist, when meet the maxsize or add&&key exist return error
func (this *Cache) set(key string, valuePtr *interface{}, add bool) error {
	now := time.Now()
	// TODO time stamp
	this.lock.Lock()
	itemPtr, found := this.data[key]
	if !found {
		if this.size == this.maxSize {
			this.lock.Unlock()
			return errors.New("Size limit")
		}
		//create new item
		this.size++
		itemPtr = &item{}
		this.data[key] = itemPtr
		itemPtr.lock.Lock()
		this.lock.Unlock()
		itemPtr.updateTime = now
		itemPtr.expiration = now.Add(this.expiration).UnixNano()
		itemPtr.objectPtr = valuePtr
		itemPtr.lock.Unlock()
		return nil
	} else if add {
		this.lock.Unlock()
		return errors.New(key + " exist")
	}
	itemPtr.lock.Lock()
	this.lock.Unlock()
	if itemPtr.updateTime.After(now) {
		itemPtr.lock.Unlock()
		return errors.New(key + " has been update")
	}
	itemPtr.updateTime = now
	itemPtr.expiration = now.Add(this.expiration).UnixNano()
	itemPtr.objectPtr = valuePtr
	itemPtr.lock.Unlock()
	return nil
}

type pack struct {
	expiration int64
	key        string
	itemPtr    *item
}

// watch clear the item in map when it expiration is end
func (this *Cache) watch() {
	t := time.NewTicker(this.interval)
	for {
		<-t.C
		now := time.Now().UnixNano()
		for k := range this.data {
			this.lock.Lock()
			// "Inlining" of expired
			if now > this.data[k].expiration {
				go this.onEvicted(k, this.data[k].objectPtr)
				delete(this.data, k)
				this.size--
			}
			this.lock.Unlock()
		}
	}
}
