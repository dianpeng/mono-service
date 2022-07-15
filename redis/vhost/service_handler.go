package vhost

import (
	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/redis/runtime"
	"github.com/dianpeng/mono-service/util"

	"github.com/tidwall/redcon"

	"fmt"
	"strings"
	"sync"
)

const (
	eventAccept  = "redis.@accept"
	eventClose   = "redis.@close"
	eventCommand = "redis.*"
)

type servicePool struct {
	idle    []*serviceHandler
	maxSize int
	sync.Mutex
}

type serviceHandler struct {
	runtime          *runtime.Runtime
	vhost            *VHost
	activeHttpClient []*util.HClient
}

func newServicePool(cacheSize int) servicePool {
	csize := cacheSize
	if csize == 0 {
		csize = g.MaxSessionCacheSize
	}
	return servicePool{
		maxSize: csize,
	}
}

func (s *servicePool) idleSize() int {
	s.Lock()
	defer s.Unlock()
	return len(s.idle)
}

func (s *servicePool) get() *serviceHandler {
	s.Lock()
	defer s.Unlock()
	if len(s.idle) == 0 {
		return nil
	}
	idleSize := len(s.idle)
	last := s.idle[idleSize-1]
	s.idle = s.idle[:idleSize-1]
	return last
}

func (s *servicePool) put(h *serviceHandler) bool {
	s.Lock()
	defer s.Unlock()
	if len(s.idle)+1 >= s.maxSize {
		return false
	}
	s.idle = append(s.idle, h)
	return true
}

func newServiceHandler(vhost *VHost) *serviceHandler {
	h := &serviceHandler{
		runtime: runtime.NewRuntimeWithModule(vhost.Module),
		vhost:   vhost,
	}
	return h
}

func (s *serviceHandler) GetHttpClient(url string) (hpl.HttpClient, error) {
	c, err := s.vhost.clientPool.Get(url)
	if err != nil {
		return nil, err
	}
	s.activeHttpClient = append(s.activeHttpClient, &c)
	return &c, nil
}

func (s *serviceHandler) finish() {
	if s.activeHttpClient != nil {
		for _, c := range s.activeHttpClient {
			s.vhost.clientPool.Put(*c)
		}
		s.activeHttpClient = nil
	}
}

func (s *serviceHandler) err(
	c redcon.Conn,
	event string,
	err error,
) {
	c.WriteError(
		fmt.Sprintf("Runtime(%s) error: %s", event, err.Error()),
	)
}

func (s *serviceHandler) onEvent(
	conn redcon.Conn,
	cmd redcon.Command,
) {
	log := alog.NewLog(s.vhost.LogFormat)

	defer func() {
		s.vhost.uploadLog(&log, nil)
		s.finish()
	}()

	cmdVal := runtime.NewCommandVal(
		&cmd,
	)

	connVal := runtime.NewConnectionVal(
		conn,
	)

	cmdName := strings.ToUpper(string(cmd.Args[0]))
	cmdEvent := fmt.Sprintf("redis.%s", cmdName)

	var err error

	if err = s.runtime.OnInit(
		connVal,
		s,
		&log,
	); err != nil {
		s.err(
			conn,
			"@init",
			err,
		)
		return
	}

	if s.runtime.Module.HasEvent(cmdEvent) {
		if _, err = s.runtime.Emit(
			cmdEvent,
			cmdVal,
		); err != nil {
			s.err(
				conn,
				cmdEvent,
				err,
			)
			return
		}
	} else {
		if _, err = s.runtime.Emit(
			eventCommand,
			cmdVal,
		); err != nil {
			s.err(
				conn,
				cmdEvent,
				err,
			)
			return
		}
	}
}

func (s *serviceHandler) onAccept(
	conn redcon.Conn,
) bool {
	log := alog.NewLog(s.vhost.LogFormat)

	defer func() {
		s.vhost.uploadLog(&log, nil)
		s.finish()
	}()

	connVal := runtime.NewConnectionVal(
		conn,
	)

	var err error

	if err = s.runtime.OnInit(
		connVal,
		s,
		&log,
	); err != nil {
		s.err(
			conn,
			"@init",
			err,
		)
		return false
	}

	if val, err := s.runtime.Emit(
		eventAccept,
		pl.NewValNull(),
	); err != nil {
		s.err(
			conn,
			eventAccept,
			err,
		)
		return false
	} else {
		if val.IsBool() {
			return val.Bool()
		} else {
			return true
		}
	}
}

func (s *serviceHandler) onClose(
	conn redcon.Conn,
	connErr error,
) {
	log := alog.NewLog(s.vhost.LogFormat)

	defer func() {
		s.vhost.uploadLog(&log, nil)
		s.finish()
	}()

	connVal := runtime.NewConnectionVal(
		conn,
	)

	var err error
	ctx := pl.NewValNull()
	if connErr != nil {
		ctx = pl.NewValStr(connErr.Error())
	}

	if err = s.runtime.OnInit(
		connVal,
		s,
		&log,
	); err != nil {
		s.err(
			conn,
			"@init",
			err,
		)
		return
	}

	if _, err := s.runtime.Emit(
		eventClose,
		ctx,
	); err != nil {
		s.err(
			conn,
			eventClose,
			err,
		)
	}
}
