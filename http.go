package cache_go
import (
	"cache-go/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

/*
	http:
		处理客户端的请求
		客户端发送的请求
*/

const(
	defaultBasePath = "/cache/"
	defaultReplicas = 50
)
type HTTPPool struct {
	self 	 string  	// 自己的地址
	basePath string    // 前缀
	mu  sync.Mutex
	peers  *consistenthash.Map // 一致性哈希算法
	httpGetters map[string]*httpGetter // 远程节点地址映射表
}

type httpGetter struct {
	baseURL string    // 要访问的远程节点地址
}

// NewHTTpPool 创建一个客户端请求
func NewHTTpPool(self string) *HTTPPool  {
	return &HTTPPool{
		self: self,
		basePath: defaultBasePath,
	}
}

// Log 实现客户端的log日志
func (p *HTTPPool) Log(format string,v...interface{}){
	log.Printf("[Server %s] %s",p.self,fmt.Sprintf(format,v...))
}

// ServerHTTP 给客户端的响应.与客户端实现交互服务
func (p *HTTPPool) ServerHTTP(w http.ResponseWriter,r *http.Request)  {
	/**
	根据url获取缓存空间以及缓存数据,响应给前端
	*/
	// 请求前缀处理
	if !strings.HasPrefix(r.URL.Path,p.basePath) {
		panic("HTTPPool serving unexpected path:"+ r.URL.Path)
	}
	p.Log("%s %s",r.Method,r.URL.Path)

	//	 处理请求,从请求路径中获取缓存空间的名字以及key
	parts := strings.SplitN(r.URL.Path[len(p.basePath):],"/",2)
	if len(parts) != 2 {
		http.Error(w,"bad request",http.StatusBadRequest)
		return
	}

	//	 根据缓存空间名字,获取对应的缓存空间
	groupName := parts[0]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w,"no such group :" + groupName,http.StatusNotFound)
		return
	}

	//	 根据key获取对应的value
	key := parts[1]
	view,err := group.Get(key)
	if err != nil {
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type","application/octet-stream")
	//  写入http响应体中
	w.Write(view.ByteSlice())
}

// PickPeer 访问远程节点,获取缓存数据
func (p *HTTPPool) PickPeer(key string) (PeerGetter,bool){
	p.mu.Lock()
	defer p.mu.Unlock()
	// 通过key获取远程节点,得到该节点对应的url
	if peer := p.peers.Get(key); peer != "" && peer != p.self{
		p.Log("Pick peer %s",peer)
		return  p.httpGetters[peer],true
	}
	return nil,false
}

// Set 设置真实节点以及虚拟节点
func (p *HTTPPool) Set(peers ...string){
	p.mu.Lock()
	defer p.mu.Unlock()
	//	调用一致性哈希算法,添加真实节点和虚拟节点
	p.peers = consistenthash.New(defaultReplicas,nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter,len(peers))
	for _,peer := range peers{
		// 添加真实节点对应的url地址
		p.httpGetters[peer] = &httpGetter{baseURL:peer+p.basePath}
	}

}

// Get 获取某个节点中缓存空间中key对应的值 : 这里是通过拼接路径的方式获取
func (h *httpGetter) Get(group string,key string)([]byte,error)  {
	u := fmt.Sprintf(
		"%v%v%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// 发起http请求
	res,err := http.Get(u)
	if err != nil {
		return nil,err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK{
		return nil,fmt.Errorf("server returned:%v",res.Status)
	}
	bytes,err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil,fmt.Errorf("reading response body:%v",err)
	}
	return  bytes,nil

}


/**
用户角度出发:
	1.根据key从缓存中拿数据
		分布式场景中:
				1.根据key获取对应的节点
				2.获取对应节点的url
				3.拿着url重新访问真实节点的缓存
*/