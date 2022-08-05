package cache_go

import (
	"cache-go/singleflight"
	"fmt"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte,error) // 回调函数,获取数据源
}

type GetterFunc func(key string)([]byte,error) // 函数类型,实现上面的接口

// Get 实现Getter接口,返回内容为本身
func (f GetterFunc) Get(key string) ([]byte,error){
	return f(key)
}

type Group struct {
	name  string   // 缓存名字
	getter Getter // 获取数据源
	mainCache cache // 缓存
	peers PeerPicker // 分布式节点
	loader *singleflight.Group // 预防缓存击穿
}

var(
	mu sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string,cacheBytes int64,getter Getter) *Group  {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		getter: getter,
		mainCache: cache{cacheBytes:cacheBytes},
		loader: &singleflight.Group{},
	}
//	 存入map
	groups[name] = g
	return g
}

// RegisterPeers 注册节点
func (g *Group) RegisterPeers(peers PeerPicker)  {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// GetGroup 根据名字切换到对应的缓冲空间
func GetGroup(name string) *Group {
	// 这里使用了读锁
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 根据key获取对应缓存空间的值
func (g *Group) Get(key string)(ByteView,error){
	if key == ""{
		return ByteView{},fmt.Errorf("key is required")
	}
	// 直接从缓存中获取
	if v,ok := g.mainCache.get(key);ok{
		log.Println("[Cache] hit")
		return v,nil
	}
//	不存在,去其他节点或者db中获取
	return g.load(key)
}

func (g *Group) load(key string)(value ByteView,err error)  {
	// 阻拦多次请求,防止缓存击穿
	viewi,err := g.loader.Do(key, func()(interface{},error) {
		if g.peers != nil {
			// 选择节点
			if peer,ok := g.peers.PickPeer(key);ok {
				// 从分布式节点中获取值
				if value,err = g.getFromPeer(peer,key);err == nil{
					return value,nil
				}
				log.Println("[GeeCache] Failed to get from peer",err)
			}
		}
	//	否则,调取其他资源,一般只db
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView),nil
	}
	return
}

func (g *Group) getLocally(key string)(ByteView,error)  {
	// 获取数据源
	bytes,err := g.getter.Get(key)
	if err != nil {
		return ByteView{},err
	}
	// 这里用拷贝是因为bytes是切片,即引用类型
	value := ByteView{b:cloneBytes(bytes)}
	// 放入缓存中
	g.populateCache(key,value)
	return value,nil
}
func (g *Group) populateCache(key string,value ByteView)  {
	g.mainCache.add(key,value)
}

// getFromPeer 从节点中获取值
func (g *Group) getFromPeer(peer PeerGetter,key string)(ByteView,error)  {
	bytes,err := peer.Get(g.name,key)
	if err != nil {
		return ByteView{},err
	}
	return ByteView{b:bytes},nil
}

/**
	本程序主要实现了缓存的调用以及访问,同时预防缓存击穿和穿透的方法
*/

