package memory_cache

import (
	"sync"
	"time"
	"memory_cache/list"
)

//--------------------------------------
// item views
//--------------------------------------

type ItemViews struct {
	key          string
	view         int
	lastViewTime time.Time
}

func NewItemViews(key string) *ItemViews {
	return &ItemViews{
		key:          key,
		view:         1,
		lastViewTime: time.Now(),
	}
}

func (this *ItemViews) Destory(i interface{}) {}

func (this *ItemViews) Equal(i1 interface{}, i2 interface{}) bool {
	if _, ok := i1.(*ItemViews); !ok {
		return false
	}

	if _, ok := i2.(*ItemViews); !ok {
		return false
	}

	return i1.(*ItemViews).key == i2.(*ItemViews).key
}

func (this *ItemViews) UpdateWithExpireExam(d time.Duration) (view int, expired bool) {
	if expired = this.expired(d); expired {
		view = this.view
		return
	}
	this.view++
	view = this.view
	this.lastViewTime = time.Now()
	return
}

func (this *ItemViews) expired(d time.Duration) bool {
	if time.Now().Sub(this.lastViewTime) > d {
		this.view = 1
		return true
	}
	return false
}

func (this *ItemViews) Reset() *ItemViews {
	this.view = 1
	this.lastViewTime = time.Now()
	return this
}

//--------------------------------------
// cache item
//--------------------------------------

type CacheItem struct {
	data  interface{}
	views *ItemViews
}

func (this *CacheItem) Destory(i interface{}) {}

func (this *CacheItem) Equal(i1 interface{}, i2 interface{}) bool {
	if _, ok := i1.(*CacheItem); !ok {
		return false
	}

	if _, ok := i2.(*CacheItem); !ok {
		return false
	}

	return i1.(*CacheItem).views.key == i2.(*CacheItem).views.key
}

func NewCacheItem(views *ItemViews, d interface{}) *CacheItem {
	return &CacheItem{
		data:  d,
		views: views,
	}
}

//--------------------------------------
// view history
//--------------------------------------

type ViewHistory struct {
	k      int
	size   int
	items  *list.List
	expire time.Duration
}

func NewViewHistory(size int, expire time.Duration, k int) *ViewHistory {
	return &ViewHistory{
		k:      k,
		size:   size,
		items:  list.NewList(&ItemViews{}),
		expire: expire,
	}
}

func (this *ViewHistory) Put(key string) (new *list.Element, old string) {
	view := NewItemViews(key)
	new = this.items.Insert(nil, view)
	if this.items.Size > this.size {
		old = this.items.Tail.Data.(*ItemViews).key
		this.items.Delete(this.items.Tail)
	}
	return
}

func (this *ViewHistory) UpdateView(pointer *list.Element) (cached bool) {
	this.items.Delete(pointer)
	if views, _ := pointer.Data.(*ItemViews).UpdateWithExpireExam(this.expire); views > this.k {
		cached = true
		return
	}
	this.items.Insert(nil, pointer)
	return
}

//--------------------------------------
// memory cache
//--------------------------------------

type MemoryCache struct {
	size   int
	items  *list.List
	expire time.Duration
}

func NewMemoryCache(size int, expire time.Duration) *MemoryCache {
	return &MemoryCache{
		size:   size,
		expire: expire,
		items:  list.NewList(&CacheItem{}),
	}
}

func (this *MemoryCache) Put(i *ItemViews, data interface{}) (new *list.Element, old string) {
	new = this.items.Insert(nil, NewCacheItem(i.Reset(), data))
	if this.items.Size > this.size {
		old = this.items.Tail.Data.(*CacheItem).views.key
		this.items.Delete(this.items.Tail)
	}
	return
}

func (this *MemoryCache) UpdateData(pointer *list.Element, data interface{}) {
	pointer.Data.(*CacheItem).data = data
	this.items.Delete(pointer)
	this.items.Insert(nil, pointer)
}

func (this *MemoryCache) UpdateView(pointer *list.Element) (expired bool) {
	this.items.Delete(pointer)
	if _, expired = pointer.Data.(*CacheItem).views.UpdateWithExpireExam(this.expire); expired {
		return
	}
	this.items.Insert(nil, pointer)
	return
}

//--------------------------------------
// LRU-K memory cache manager
//--------------------------------------

type MemoryCacheManager struct {
	history     *ViewHistory
	cache       *MemoryCache
	historyMap  *sync.Map
	cacheMap    *sync.Map
	historyLock *sync.Mutex
	cacheLock   *sync.Mutex
}

func NewMemoryCacheManager(historySize int, hisotrySatisfyK int, historyExpire time.Duration, memoryCacheSize int, memoryCacheExpire time.Duration) *MemoryCacheManager {
	return &MemoryCacheManager{
		history:     NewViewHistory(historySize, historyExpire, hisotrySatisfyK),
		cache:       NewMemoryCache(memoryCacheSize, memoryCacheExpire),
		historyMap:  &sync.Map{},
		cacheMap:    &sync.Map{},
		historyLock: &sync.Mutex{},
		cacheLock:   &sync.Mutex{},
	}
}

func (this *MemoryCacheManager) Get(key string) (data interface{}) {
	this.historyLock.Lock()
	defer this.historyLock.Unlock()
	if val, ok := this.historyMap.Load(key); ok {
		if cached := this.history.UpdateView(val.(*list.Element)); cached {
			n, old := this.cache.Put(val.(*list.Element).Data.(*ItemViews), nil)
			this.historyMap.Delete(key)
			this.cacheMap.Store(key, n)
			if len(old) != 0 {
				this.cacheMap.Delete(old)
			}
		}
	} else if val, ok = this.cacheMap.Load(key); ok {
		expire := time.Now().Add(500 * time.Millisecond)
		for {
			if time.Now().After(expire) {
				break
			}
			if data = val.(*list.Element).Data.(*CacheItem).data; data != nil {
				break
			}
		}
		this.cacheLock.Lock()
		defer this.cacheLock.Unlock()
		if this.cache.UpdateView(val.(*list.Element)) {
			this.cacheMap.Delete(key)
			return
		}
	} else {
		n, old := this.history.Put(key)
		this.historyMap.Store(key, n)
		if len(old) != 0 {
			this.historyMap.Delete(old)
		}
	}
	return
}

func (this *MemoryCacheManager) Put(key string, data interface{}) {
	this.cacheLock.Lock()
	defer this.cacheLock.Unlock()
	var (
		ok  bool
		val interface{}
	)
	if val, ok = this.cacheMap.Load(key); !ok {
		return
	}
	this.cache.UpdateData(val.(*list.Element), data)
}



