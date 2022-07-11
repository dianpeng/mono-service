package vhost

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/hclient"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/http/phase"
	"github.com/dianpeng/mono-service/http/runtime"
	"github.com/dianpeng/mono-service/pl"
)

type servicePool struct {
	idle    []*serviceHandler
	maxSize int
	sync.Mutex
}

type serviceHandler struct {
	service          *framework.Service
	runtime          *runtime.Runtime
	vhs              *vHS
	activeHttpClient []*hclient.HClient

	// result generated by service.Accept function which will be used during
	// phase response
	serviceResult framework.ApplicationResult
	phase         string
	phaseIndex    int
}

func newServicePool(cacheSize int) servicePool {
	cache_size := cacheSize
	if cache_size == 0 {
		cache_size = g.MaxSessionCacheSize
	}
	return servicePool{
		maxSize: cache_size,
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

func newServiceHandler(s *framework.Service, vhs *vHS) *serviceHandler {
	h := &serviceHandler{
		service: s,
		runtime: runtime.NewRuntimeWithModule(vhs.module),
		vhs:     vhs,
	}
	return h
}

// implementation of serviceHandler --------------------------------------------
func (s *serviceHandler) setPhase(p int, n string) {
	s.phaseIndex = p
	s.phase = n
}

// entry function for performing one http transactions until we are done
func (s *serviceHandler) main(
	req *http.Request,
	p hrouter.Params,
	resp http.ResponseWriter,
	prevError error,
) {
	var applicationContext interface{}

	reqVal := hpl.NewRequestVal(req)

	routerVal := hpl.NewRouterParamsVal(p)

	respWrapper, respVal := newResponseWriterWrapper(
		s,
		resp,
	)

	log := alog.NewLog(s.vhs.vhost.LogFormat)
	logP := &logProvider{
		s: s,
	}

	defer func() {

		// (4) finalize the response ie flushing them out. Notes this MUST be
		//     before the application.Done
		{
			s.setPhase(phase.PhaseHttpResponseFinalize, "http.response_finalize")
			respWrapper.Finalize()
		}

		// (5) finalize the application handler with application done
		if applicationContext != nil {
			s.setPhase(phase.PhaseApplicationDone, "application.done")
			s.service.App.Done(applicationContext)
		}

		// (6) finally, generate the access log
		{
			s.setPhase(phase.PhaseAccessLog, ".access_log")
			s.Log(&log)
		}

		// cleanup work
		s.vhs.vhost.uploadLog(
			&log,
			logP,
		)
		s.finish()
		s.vhs.servicePool.put(s)
	}()

	// (0) run the HPL session init
	{
		s.setPhase(phase.PhaseInit, ".init")
		if err := s.init(reqVal, routerVal, respVal, &log); err != nil {
			respWrapper.ReplyErrorHPL(err)
			return
		}

		// extream cases, that the initialization phase of session will just
		// terminate the execution of the response
		if respWrapper.IsFlushed() {
			return
		}
	}

	if prevError != nil {
		s.setPhase(phase.PhaseCreateService, ".create_service")
		respWrapper.ReplyErrorCreateService(
			prevError,
		)
		return
	}

	// (1) run the request middleware phase
	s.setPhase(phase.PhaseHttpRequest, "http.request")
	if !s.service.Request.Accept(
		req,
		p,
		respWrapper,
		s,
	) {
		return
	}

	// (2) run the application
	{
		app := s.service.App

		s.setPhase(phase.PhaseApplicationPrepare, "application.prepare")
		context, err := app.Prepare(
			req,
			p,
		)
		if err != nil {
			respWrapper.ReplyErrorAppPrepare(
				err,
			)
			return
		}

		applicationContext = context
		s.setPhase(phase.PhaseApplicationAccept, "application.accept")
		r, err := app.Accept(
			context,
			s,
		)

		if err != nil {
			respWrapper.ReplyErrorAppAccept(
				err,
			)
			return
		}

		s.serviceResult = r
		// before enter into user's response middleware, run application generated
		// event if applicable
		if s.serviceResult.Event != "" {
			s.setPhase(phase.PhaseApplicationEvent, "application.event")
			if err := s.runtime.RunWithContext(s.serviceResult.Event,
				s.serviceResult.Context); err != nil {
				respWrapper.ReplyErrorHPL(err)
				return
			}
		}
	}

	// (3) run the middleware response phase, notes the response middleware
	//     execution is different from running request middleware
	{
		s.setPhase(phase.PhaseHttpResponse, "http.response")

		// start to run response middleware chain
		if !s.service.Response.Accept(
			req,
			p,
			respWrapper,
			s,
		) {
			return
		}
	}
}

func (s *serviceHandler) Log(log *alog.Log) error {
	return s.runtime.RunWithContext(EventNameLog, pl.NewValNull())
}

// interface for frame.ServiceContext
func (s *serviceHandler) Runtime() *runtime.Runtime {
	return s.runtime
}

func (s *serviceHandler) HplSessionWrapper() runtime.SessionWrapper {
	return s
}

// interface for alog.ServiceInfo
func (s *serviceHandler) ServiceName() string {
	return s.vhs.config.Name
}

func (s *serviceHandler) SessionPhase() string {
	return s.phase
}

func (s *serviceHandler) init(
	request pl.Val,
	hrouter pl.Val,
	response pl.Val,
	log *alog.Log,
) error {

	return s.runtime.OnInit(
		request,
		hrouter,
		response,
		s,
		log,
	)
}

func (s *serviceHandler) finish() {
	// http client pool draining operations
	if s.activeHttpClient != nil {
		for _, c := range s.activeHttpClient {
			s.vhs.vhost.clientPool.Put(*c)
		}
	}
}

// interface for hpl.SessionWrapper
func (s *serviceHandler) GetHttpClient(url string) (hpl.HttpClient, error) {
	c, err := s.vhs.vhost.clientPool.Get(url)
	if err != nil {
		return nil, err
	}
	s.activeHttpClient = append(s.activeHttpClient, &c)
	return &c, nil
}

func (s *serviceHandler) OnLoadVar(_ *pl.Evaluator, name string) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf(
		"unknown variable name: %s",
		name,
	)
}

func (s *serviceHandler) OnStoreVar(_ *pl.Evaluator, name string, _ pl.Val) error {
	return fmt.Errorf(
		"unknown variable name: %s",
		name,
	)
}

func (s *serviceHandler) OnAction(_ *pl.Evaluator, name string, _ pl.Val) error {
	return fmt.Errorf(
		"unknown action name: %s",
		name,
	)
}

func (s *serviceHandler) GetPhaseName() string {
	return s.phase
}
