package hservice

import (
	"github.com/dianpeng/mono-service/config"
	_ "github.com/dianpeng/mono-service/module"
	"sync"
)

type HService struct {
	VHostList []*VHost
	wg        *sync.WaitGroup
}

func NewHService(config *config.Config) (*HService, error) {
	var vlist []*VHost
	for _, vhost := range config.VHostList {
		v, err := newVHost(vhost)
		if err != nil {
			return nil, err
		}
		vlist = append(vlist, v)
	}
	return &HService{
		VHostList: vlist,
		wg:        &sync.WaitGroup{},
	}, nil
}

// try to run all the VHost registered
func (h *HService) Run() {
	h.wg.Add(len(h.VHostList))

	for _, vv := range h.VHostList {
		go func() {
			defer h.wg.Done()
			vv.Run()
		}()
	}

	h.wg.Wait()
}
