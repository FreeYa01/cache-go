package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32  // hash 算法

type Map struct {
	hash   			Hash  // 哈希函数
	replicas 		int   // 虚拟节点容量
	keys          []int   // 存入的虚拟节点
	hashMap       map[int]string // 真实节点和虚拟节点的映射,用map实现
}

func New(replicas int, fn Hash) *Map {
	m  := &Map{
		replicas: replicas,   // 虚拟节点数量
		hash: fn,			 // hash函数值
		hashMap: make(map[int]string),
	}
	// 默认hash规则
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}
// Add 添加节点
func (m *Map) Add(keys ...string){
	for _,key := range keys {
		for i := 0; i < m.replicas ; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i)+key)))
			// 添加到hash环上
			m.keys = append(m.keys,hash)
		   // 添加虚拟节点对应的真实节点
		   	m.hashMap[hash] = key
		}
	}
	// 对哈希环上的值进行递增排序,保证后序顺时针查找时候方便
	sort.Ints(m.keys)
}

// Get 根据key在哈希环上找到对应的节点
func (m *Map) Get(key string) string  {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))

//	 在环上顺时针寻找第一个大于改key的节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
//	返回真实节点:记得求余
	return m.hashMap[m.keys[idx % len(m.keys)]]
}