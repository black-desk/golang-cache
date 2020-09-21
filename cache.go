// package cache provide a thread-safe KV-container
package cache

import (
	"errors"
	"sync"
	"time"
)

// Cache is a KV-container
type Cache struct {
	expiration time.Duration              // expiration time for all key-value pair store in this container
	data       map[string]*item           // a map hold the pointer to item
	lock       sync.RWMutex               // a lock for the map above
	size       uint32                     // current amount of items in this Cache
	maxSize    uint32                     // the max amount of items Cache can contain
	onEvicted  func(string, *interface{}) // a function to call when item evicted
	queue      chan pair                  // use chan to emulate a queue in which the items sort in ascending order of expiration, see detail in Cache.watch()
	lookat     pair                       // the head of queue
}

// Get a new Cache, return the pointer to it
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

// Get the value bind to the key
func (this *Cache) Get(key string) (interface{}, error) {
	itemPtr, found := this.getItemPtr(key, true)
	if !found {
		return nil, errors.New(key + " not found!")
	}
	return *(itemPtr.getObjectPtr()), nil
}

// get a pointer which point to the item bind with the key, lock means this function should lock the Cache.data map or not
func (this *Cache) getItemPtr(key string, lock bool) (*item, bool) {
	if lock {
		this.lock.RLock()
	}
	targetItem, found := this.data[key]
	if lock {
		this.lock.RUnlock()
	}
	return targetItem, found
}

// item hold a pointer to the actual object, the lock is for the ptr, cnt count the pointer point to this item in queue
type item struct {
	expiration int64
	lock       sync.RWMutex
	cnt        uint32
	objectPtr  *interface{}
}

// get the pointer, Rlock for concurrent set
func (this *item) getObjectPtr() *interface{} {
	this.lock.RLock()
	p := this.objectPtr
	this.lock.RUnlock()
	return p
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
	this.lock.Lock()
	it, found := this.getItemPtr(key, false)
	if !found {
		if this.size == this.maxSize {
			return errors.New("Size limit")
		}
		this.size++
		it = &item{}
		this.data[key] = it
	} else if add {
		return errors.New(key + " exist")
	}
	it.lock.Lock()
	this.lock.Unlock()
	it.objectPtr = valuePtr
	it.cnt++
	it.expiration = time.Now().Add(this.expiration).UnixNano()
	it.lock.Unlock()
	this.queue <- pair{key, it} // push back to queue there, which means that the create time of elements in the queue is ascending
	return nil
}

type pair struct {
	key  string
	item *item
}

// watch clear the item in map when it expiration is end
func (this *Cache) watch() {
	this.lookat = <-this.queue
	for {
		_item := this.lookat.item
		_key := this.lookat.key
		_item.lock.Lock()
		if _item.cnt > 1 { // if cnt > 1 it means after this item push in this queue, it have been set again, so there is another pair has a pointer point to this item after this _item in queue. which means we have nothing to do wich the _item
			_item.cnt--
		} else if _item.expiration < time.Now().UnixNano() {
			this.lock.Lock()
			delete(this.data, _key)
			this.lock.Unlock()
			go this.onEvicted(_key, _item.objectPtr)
		}
		this.lookat = <-this.queue
		_item.lock.Unlock()
	}
}
