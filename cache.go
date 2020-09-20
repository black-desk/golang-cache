package cache

import (
	"errors"
	"sync"
	"time"
)

type item struct {
	expiration int64
	lock       sync.RWMutex
	cnt        uint32
	objectPtr  *interface{}
}

func (this *item) getObjectPtr() *interface{} {
	this.lock.RLock()
	p := this.objectPtr
	this.lock.RUnlock()
	return p
}

type pair struct {
	key  string
	item *item
}

type Cache struct {
	expiration time.Duration
	data       map[string]*item
	lock       sync.RWMutex
	size       uint32
	maxSize    uint32
	onEvicted  func(string, *interface{})
	lookat     pair
	queue      chan pair
}

func NewCache(expiration time.Duration, maxSize uint32, onEvicted func(string, *interface{})) *Cache {
	c := &Cache{
		expiration: expiration,
		data:       map[string]*item{},
		lock:       sync.RWMutex{},
		size:       0,
		maxSize:    maxSize,
		onEvicted:  onEvicted,
		queue:      make(chan pair, maxSize),
	}
	go c.watch()
	return c
}

func (this *Cache) watch() {
	this.lookat = <-this.queue
	for {
		_item := this.lookat.item
		_item.lock.Lock()
		if _item.cnt > 1 {
			_item.cnt--
			this.lookat = <-this.queue
		} else if _item.expiration < time.Now().UnixNano() {
			this.lock.Lock()
			delete(this.data, this.lookat.key)
			this.lock.Unlock()
			this.lookat = <-this.queue
		}
		_item.lock.Unlock()
	}
}

func (this *Cache) Get(key string) (interface{}, error) {
	targetItem, found := this.getTargetItem(key, true)
	if !found {
		return nil, errors.New(key + " not found!")
	}
	return *(targetItem.getObjectPtr()), nil
}

func (this *Cache) getTargetItem(key string, lock bool) (*item, bool) {
	if lock {
		this.lock.RLock()
	}
	targetItem, found := this.data[key]
	if lock {
		this.lock.RUnlock()
	}
	return targetItem, found
}

func (this *Cache) Set(key string, value interface{}) error {
	valuePtr := &value
	return this.set(key, valuePtr, false)
}

func (this *Cache) Add(key string, value interface{}) error {
	valuePtr := &value
	return this.set(key, valuePtr, true)
}

func (this *Cache) set(key string, valuePtr *interface{}, add bool) error {
	this.lock.Lock()
	targetItem, found := this.getTargetItem(key, false)
	if !found {
		if this.size == this.maxSize {
			return errors.New("Size limit")
		}
		this.size++
		targetItem = &item{}
		this.data[key] = targetItem
	} else if add {
		return errors.New(key + " exist")
	}
	targetItem.lock.Lock()
	this.lock.Unlock()
	targetItem.objectPtr = valuePtr
	targetItem.cnt++
	targetItem.expiration = time.Now().Add(this.expiration).UnixNano()
	targetItem.lock.Unlock()
	this.queue <- pair{key, targetItem}
	return nil
}
