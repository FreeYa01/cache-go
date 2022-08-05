package cache_go
/**
	分布式节点的实体
*/

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter,ok bool) // 选择哈希环上相应的节点,并获取该节点中相应的值
}
type PeerGetter interface {
	Get(group string,key string)([]byte,error)  // 获取到达缓存空间的路径,从该路径中获取缓存值
}