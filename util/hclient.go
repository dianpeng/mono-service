package util

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type HClient struct {
	Client *http.Client
	URL    *url.URL
	req    *http.Request
	resp   *http.Response
	err    error
}

func (h *HClient) Do(req *http.Request) (*http.Response, error) {
	h.req = req
	resp, err := h.Client.Do(req)
	if err != nil {
		h.err = err
	} else {
		h.err = nil
		h.resp = resp
	}

	return resp, err
}

type clientList []HClient
type pool map[string]clientList

func cacheKey(url *url.URL) string {
	return url.Scheme + ":" + url.Host
}

func (h *HClient) CacheKey() string {
	return cacheKey(h.URL)
}

// A simple thread safe client(http) pool. Each Vhost object will have its own
// hclient object which manage a list of http client and we can perform active
// health check on top of it as well
type HClientPool struct {
	Name          string
	p             pool
	size          int64
	drain         chan HClient
	drainSize     int64
	maxPoolSize   int64
	maxDrainSize  int64
	clientTimeout int64
	reuseSize     int64
	newSize       int64
	drainProduce  int64
	drainConsume  int64
	sync.Mutex
}

func (h *HClientPool) Stats() interface{} {
	o := make(map[string]interface{})
	{
		h.Lock()
		o["name"] = h.Name
		o["size"] = h.size
		o["drainSize"] = h.drainSize
		o["maxPoolSize"] = h.maxPoolSize
		o["maxDrainSize"] = h.maxDrainSize
		o["clientTimeout"] = h.clientTimeout
		o["reuseSize"] = h.reuseSize
		o["newSize"] = h.newSize
		o["drainConsume"] = h.drainConsume
		o["drainProduce"] = h.drainProduce
		h.Unlock()
	}
	return o
}

func (h *HClientPool) tryGet(url *url.URL) HClient {
	h.Lock()
	defer h.Unlock()

	if h.size == 0 || len(h.p) == 0 {
		return HClient{}
	}

	key := cacheKey(url)

	x, ok := h.p[key]
	if !ok {
		return HClient{}
	}
	if len(x) == 0 {
		return HClient{}
	}

	// pop the back from the queue
	last := x[len(x)-1]
	x = x[:len(x)-1]
	h.p[key] = x

	h.reuseSize++
	h.size--

	return last
}

func (h *HClientPool) tryCreate(url *url.URL) (HClient, error) {
	{
		h.Lock()
		h.newSize++
		h.Unlock()
	}
	c := &http.Client{
		Timeout: time.Duration(h.clientTimeout) * time.Second,
	}
	return HClient{
		Client: c,
		URL:    url,
	}, nil
}

func (h *HClientPool) DrainSize() int64 {
	h.Lock()
	defer h.Unlock()
	return h.drainSize
}

func (h *HClientPool) CacheSize() int64 {
	h.Lock()
	defer h.Unlock()
	return h.size
}

func (h *HClientPool) Get(rawStr string) (HClient, error) {
	url, err := url.Parse(rawStr)
	if err != nil {
		return HClient{}, err
	}
	switch url.Scheme {
	case "http", "https":
		break
	default:
		return HClient{}, fmt.Errorf("URL: %s unsupported scheme", rawStr)
	}

	if c := h.tryGet(url); c.Client != nil {
		return c, nil
	} else {
		return h.tryCreate(url)
	}
}

func (h *HClientPool) tryDrain(c HClient) {
	if c.resp == nil {
		return
	}

	{
		h.Lock()
		h.drainSize++
		h.drainProduce++
		h.Unlock()
	}
	h.drain <- c
}

func (h *HClientPool) shouldPut() bool {
	h.Lock()
	defer h.Unlock()
	return h.size+h.drainSize+1 < h.maxPoolSize
}

func (h *HClientPool) Put(c HClient) bool {
	if h.shouldPut() {
		h.tryDrain(c)
		return true
	}
	return false
}

func (h *HClientPool) putBack(x HClient) {
	h.Lock()
	h.drainSize--
	h.size++

	ckey := x.CacheKey()
	l, ok := h.p[ckey]
	if ok {
		l = append(l, x)
	} else {
		l = []HClient{x}
	}
	h.p[ckey] = l
	h.Unlock()
}

func (h *HClientPool) doDrain(max int64) {
	for {
		x := <-h.drain
		{
			h.Lock()
			h.drainConsume++
			h.Unlock()
		}

		if x.resp == nil {
			continue
		}
		_, err := io.CopyN(io.Discard, x.resp.Body, max)
		x.resp.Body.Close()

		if err != nil && err != io.EOF {
			continue
		}

		h.putBack(x)
	}

	panic("never reach here")
}

func NewHClientPool(name string, maxPoolSize int64, clientTimeout int64, maxDrain int64) *HClientPool {
	c := &HClientPool{
		Name:          name,
		p:             make(pool),
		size:          int64(0),
		drain:         make(chan HClient),
		drainSize:     int64(0),
		maxPoolSize:   maxPoolSize,
		maxDrainSize:  maxDrain,
		clientTimeout: clientTimeout,
	}

	go c.doDrain(maxDrain)
	return c
}
