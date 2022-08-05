package singleflight

import "sync"

type call struct {
	wg sync.WaitGroup
	val interface{}
	err error
}
type Group struct {
	mu  sync.Mutex
	m map[string]*call
}

func (g *Group) Do(key string,fn func()(interface{},error)) (interface{},error)  {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 当前key获取了任务
	if c,ok := g.m[key]; ok {
		g.mu.Unlock()
		// 其他任务进行等待
		c.wg.Wait()
		return c.val,c.err
	}
	//  创建一个请求
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val,c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	// 删除请求
	delete(g.m,key)
	g.mu.Unlock()
	return c.val,c.err

}
