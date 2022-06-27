package server

import (
	_ "github.com/dianpeng/mono-service/application"
	"sync"
)

type HService struct {
	VHostList []*vhost
	wg        *sync.WaitGroup
}

func NewHService(pathList []string) (*HService, error) {
	var vlist []*vhost

	for _, p := range pathList {
		vhs, err := createVHost(
			p,
		)
		if err != nil {
			return nil, err
		}
		vlist = append(vlist, vhs)
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
