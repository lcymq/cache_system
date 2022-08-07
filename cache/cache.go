package cache

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	NoExpiration      time.Duration = -1 // 没有过期的时间标志, 代表数据项永远不过期
	DefaultExpiration time.Duration = 0  // 默认的过期时间, 标记数据项应该拥有一个默认过期时间
)

type Item struct {
	Object     interface{} // 存储任意类型的对象
	Expiration int64       // 数据项过期时间，Unix时间戳，单位是纳秒
}

// 判断数据项是否已经过期
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type Cache struct {
	DefaultExpiration time.Duration
	items             map[string]Item // 缓存数据项存在map中
	mu                sync.RWMutex    // 读写锁
	gcInterval        time.Duration   // 过期数据项清理周期
	stopGC            chan bool
}

// 过期缓存数据项清理
/*
通过 time.NewTicker() 方法创建的 ticker,
会通过指定的c.Interval 间隔时间，
周期性的从 ticker.C 管道中发送数据过来，
我们可以根据这一特性周期性的执行 DeleteExpired() 方法。
*/

/*
为使 gcLoop()函数能正常结束，
我们通过监听c.stopGc管道，
如果有数据从该管道中发送过来，
我们就停止 gcLoop() 的运行。
*/
func (c *Cache) gcLoop() {
	ticker := time.NewTicker(c.gcInterval) //
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired() // 通过time.Ticker定期执行DeleteExpired()方法，清理过期的数据项
		case <-c.stopGC:
			ticker.Stop()
			return
		}
	}
}

// 删除缓存数据项
func (c *Cache) delete(k string) {
	delete(c.items, k)
}

// 删除过期数据项
func (c *Cache) DeleteExpired() {
	now := time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.items {
		if v.Expiration > 0 && now > v.Expiration {
			c.delete(k)
		}
	}
}

// 设置缓存数据项，如果数据项存在则覆盖
func (c *Cache) Set(k string, v interface{}, d time.Duration) {
	var e int64
	if d == DefaultExpiration {
		d = c.DefaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[k] = Item{
		Object:     v,
		Expiration: e,
	}
}

// 设置数据项，没有锁操作
func (c *Cache) set(k string, v interface{}, d time.Duration) {
	var e int64
	if d == DefaultExpiration {
		d = c.DefaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.items[k] = Item{
		Object:     v,
		Expiration: e,
	}
}

// 获取数据项，如果找到数据项，还需要判断数据项是否已经过期
func (c *Cache) get(k string) (interface{}, bool) {
	item, found := c.items[k]
	if !found {
		return nil, false
	}
	if item.Expired() {
		return nil, false
	}
	return item.Object, true
}

// 添加数据项，如果数据已经存在，则返回错误
func (c *Cache) Add(k string, v interface{}, d time.Duration) error {
	c.mu.Lock()
	_, found := c.get(k)
	if found {
		c.mu.Unlock()
		return fmt.Errorf("Item %s already exists", k)
	}
	c.set(k, v, d)
	c.mu.Unlock()
	return nil
}

// 获取数据项
func (c *Cache) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[k]
	if !found {
		return nil, false
	}
	if item.Expired() {
		return nil, false
	}
	return item.Object, true
}

// 替换一个已经存在的数据项
func (c *Cache) Replace(k string, v interface{}, d time.Duration) error {
	c.mu.Lock()
	_, found := c.get(k)
	if !found {
		c.mu.Unlock()
		return fmt.Errorf("Item %s doesn't exist", k)
	}
	c.set(k, v, d)
	c.mu.Unlock()
	return nil
}

// 删除一个数据项
func (c *Cache) Delete(k string) {
	c.mu.Lock()
	c.delete(k)
	c.mu.Unlock()
}

// 将缓存数据项写入到io.Writer中
func (c *Cache) Save(w io.Writer) (err error) {
	enc := gob.NewEncoder(w)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Error registering from types with Gob library")
		}
	}()
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, v := range c.items {
		gob.Register(v.Object)
	}
	err = enc.Encode(&c.items)
	return
}

// 保存数据项到文件中
func (c *Cache) SaveToFile(file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	if err = c.Save(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// 从io.Reader中读取数据项
func (c *Cache) Load(r io.Reader) error {
	dec := gob.NewDecoder(r)
	items := map[string]Item{}
	err := dec.Decode(&items)
	if err == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		for k, v := range items {
			ov, found := c.items[k]
			if !found || ov.Expired() {
				c.items[k] = v // 数据项不存在或失效，将数据项加入
			}
		}
	}
	return err
}

// 从文件中加载缓存数据项
func (c *Cache) LoadFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	if err = c.Load(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// 返回缓存数据想的数量
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// 清空缓存
func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[string]Item{}
}

// 停止过期缓存清理
func (c *Cache) StopGC() {
	c.stopGC <- true
}

// 创建一个缓存系统
func NewCache(defaultExpiration, gcInterval time.Duration) *Cache {
	c := &Cache{
		DefaultExpiration: defaultExpiration,
		gcInterval:        gcInterval,
		items:             map[string]Item{},
		stopGC:            make(chan bool),
	}
	go c.gcLoop()
	return c
}
