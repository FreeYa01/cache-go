package lru

import "container/list"

type Cache struct {
	maxBytes  int64  //最大使用内存
	nbytes    int64 // 已使用内存
	ll  *list.List  // 双向链表
	cache map[string]*list.Element // map+双向链表
	onEvicted func(key string,value Value) //回调函数
}
type entry struct {
	key string
	value Value
}

type Value interface {
	Len() int
}

func New(maxBytes int64,onEvicted func(string,Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string)(value Value,ok bool){
	if ele,ok := c.cache[key];ok{
		c.ll.MoveToBack(ele)
	//	获取key
		kv := ele.Value.(entry)
		return kv.value,true
	}
	return
}

func (c *Cache) RemoveOldest()  {
	ele := c.ll.Front()
	if  ele != nil {
		// 从链表中移除改元素
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从map中删除改元素
		delete(c.cache,kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 添加到数据源
		if c.onEvicted != nil {
			c.onEvicted(kv.key,kv.value)
		}
	}
}

func (c *Cache) Add(key string,value Value){
	// 更新元素值
	if ele,ok := c.cache[key]; ok {
		c.ll.MoveToBack(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	}else{
		// 插入链表
		ele = c.ll.PushBack(&entry{key,value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 内存不足,进行移除
	for c.maxBytes != 0 && c.maxBytes < c.nbytes{
		c.RemoveOldest()
	}
}

