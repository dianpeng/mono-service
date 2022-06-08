package hservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dianpeng/mono-service/alog"
	cfg "github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/hclient"
	"github.com/dianpeng/mono-service/hpl"
	_ "github.com/dianpeng/mono-service/impl"
	"github.com/dianpeng/mono-service/phase"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/service"
	"github.com/dianpeng/mono-service/util"
	hrouter "github.com/julienschmidt/httprouter"
)

type SessionHandler interface {
	phase.PhaseAccess
	phase.PhaseRequest
	phase.PhaseResponse
	phase.PhaseLog

	CurrentSession() service.Session
}

type sessionPool struct {
	idle    []*sessionHandler
	maxSize int
	sync.Mutex
}

type vhostService struct {
	svc         service.Service
	policy      *pl.Policy
	config      *cfg.Service
	vhost       *VHost
	sessionPool sessionPool
}

type sessionHandler struct {
	session          service.Session
	hpl              *hpl.Hpl
	svc              *vhostService
	activeHttpClient []*hclient.HClient
	// result generated by session.Accept function which will be used during
	// phase response
	sessionResult    service.SessionResult
	phase            string
	phaseIndex       int
	errorDescription string
}

type VHost struct {
	Name          string
	ActiveService []*vhostService
	Router        *hrouter.Router
	HttpServer    *http.Server
	LogFormat     *alog.SessionLogFormat
	ErrorStatus   int
	ErrorBody     string
	RejectStatus  int
	RejectBody    string

	clientPool *hclient.HClientPool
}

type responseWriterWrapper struct {
	w      http.ResponseWriter
	status int
}

func newSessionPool(config *cfg.Service) sessionPool {
	cache_size := config.MaxSessionCacheSize
	if cache_size == 0 {
		cache_size = g.MaxSessionCacheSize
	}
	return sessionPool{
		maxSize: cache_size,
	}
}

func (s *sessionPool) idleSize() int {
	s.Lock()
	defer s.Unlock()
	return len(s.idle)
}

func (s *sessionPool) get() *sessionHandler {
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

func (s *sessionPool) put(h *sessionHandler) bool {
	s.Lock()
	defer s.Unlock()
	if len(s.idle)+1 >= s.maxSize {
		return false
	}
	s.idle = append(s.idle, h)
	return true
}

func (s *vhostService) getSessionHandler() (*sessionHandler, error) {
	// try to get a session handler from the pool
	if h := s.sessionPool.get(); h != nil {
		return h, nil
	}

	// create a new one
	if session, err := s.svc.NewSession(); err != nil {
		return nil, err
	} else {
		return newSessionHandler(session, s), nil
	}
}

func newvhostService(vhost *VHost, svc service.Service, config *cfg.Service) (*vhostService, error) {
	p, err := pl.CompilePolicy(config.Policy)
	if err != nil {
		return nil, err
	}
	return &vhostService{
		svc:         svc,
		policy:      p,
		config:      config,
		vhost:       vhost,
		sessionPool: newSessionPool(config),
	}, nil
}

func newSessionHandler(s service.Session, svc *vhostService) *sessionHandler {
	h := &sessionHandler{
		session: s,
		hpl:     hpl.NewHplWithPolicy(svc.policy),
		svc:     svc,
	}
	return h
}

// implementation of sessionHandler --------------------------------------------
func (s *sessionHandler) setPhase(p int, n string) {
	s.phaseIndex = p
	s.phase = n
}

func (s *sessionHandler) GetHttpClient(url string) (hpl.HttpClient, error) {
	c, err := s.svc.vhost.clientPool.Get(url)
	if err != nil {
		return nil, err
	}
	s.activeHttpClient = append(s.activeHttpClient, &c)
	return &c, nil
}

// SessionWrapper
func (s *sessionHandler) OnLoadVar(e *pl.Evaluator, name string) (pl.Val, error) {
	if s.phaseIndex == phase.PhaseHttpResponse {
		for _, x := range s.sessionResult.Vars {
			if x.Key == name {
				return x.Value, nil
			}
		}
	}

	return s.session.OnLoadVar(s.phaseIndex, e, name)
}

func (s *sessionHandler) OnStoreVar(e *pl.Evaluator, name string, v pl.Val) error {
	return s.session.OnStoreVar(s.phaseIndex, e, name, v)
}

func (s *sessionHandler) OnCall(e *pl.Evaluator, name string, args []pl.Val) (pl.Val, error) {
	return s.session.OnCall(s.phaseIndex, e, name, args)
}

func (s *sessionHandler) OnAction(e *pl.Evaluator, name string, v pl.Val) error {
	return s.session.OnAction(s.phaseIndex, e, name, v)
}

// phase handler ---------------------------------------------------------------
func (s *sessionHandler) Access(req *http.Request, p hrouter.Params) error {
	return s.hpl.OnAccess("access", req, p, s)
}

func (s *sessionHandler) Request(req *http.Request, p hrouter.Params) error {
	return s.hpl.OnRequest("request", req, p, s)
}

func (s *sessionHandler) Response(w http.ResponseWriter,
	req *http.Request,
	p hrouter.Params,
) error {
	return s.hpl.OnResponse(s.sessionResult.Event, w, req, p, s)
}

func (s *sessionHandler) Log(log *alog.SessionLog) error {
	return s.hpl.OnLog("log", log, s)
}

// interface for alog.VHostInformation
func (s *sessionHandler) VHostName() string {
	return s.svc.vhost.Name
}

func (s *sessionHandler) VHostEndpoint() string {
	return s.svc.vhost.HttpServer.Addr
}

// interface for alog.SessionInfo
func (s *sessionHandler) ServiceName() string {
	return s.session.Service().Name()
}

func (s *sessionHandler) SessionPhase() string {
	return s.phase
}

func (s *sessionHandler) ErrorDescription() string {
	return s.errorDescription
}

func (s *sessionHandler) init() error {
	return s.hpl.OnInit(s)
}

func (s *sessionHandler) finish() {
	// http client pool draining operations
	if s.activeHttpClient != nil {
		for _, c := range s.activeHttpClient {
			s.svc.vhost.clientPool.Put(*c)
		}
	}
}

// builtin handlers
func (h *VHost) httpListService(w http.ResponseWriter, _ *http.Request, _ hrouter.Params) {
	var o []interface{}

	for _, svc := range h.ActiveService {
		o = append(o, map[string]interface{}{
			"name":           svc.config.Name,
			"tag":            svc.config.Tag,
			"idl":            util.JSONSplitLine(svc.svc.IDL()),
			"policy":         util.JSONSplitLine(svc.config.Policy),
			"policyDump":     util.JSONSplitLine(svc.policy.Dump()),
			"router":         svc.config.Router,
			"methodList":     svc.config.Method,
			"idleSession":    svc.sessionPool.idleSize(),
			"maxSessionSize": svc.sessionPool.maxSize,
		})
	}

	blob, _ := json.Marshal(o)
	w.WriteHeader(200)
	w.Write(blob)
}

func (h *VHost) httpListImpl(w http.ResponseWriter, _ *http.Request, _ hrouter.Params) {
	var o []interface{}
	service.ForeachServiceFactory(
		func(svc service.ServiceFactory) {
			o = append(o, map[string]interface{}{
				"name":    svc.Name(),
				"idl":     svc.IDL(),
				"comment": util.JSONSplitLine(svc.Comment()),
			})
		},
	)

	blob, _ := json.Marshal(o)
	w.WriteHeader(200)
	w.Write(blob)
}

func (h *VHost) httpInfo(w http.ResponseWriter, _ *http.Request, _ hrouter.Params) {
	o := make(map[string]interface{})
	o["name"] = h.Name
	o["serviceNumber"] = len(h.ActiveService)
	o["endpoint"] = h.HttpServer.Addr
	o["logFormat"] = h.LogFormat.Raw
	o["errorStatus"] = h.ErrorStatus
	o["errorBody"] = h.ErrorBody
	o["rejectStatus"] = h.RejectStatus
	o["rejectBody"] = h.RejectBody
	o["httpClientPool"] = h.clientPool.Stats()

	blob, _ := json.Marshal(o)
	w.WriteHeader(200)
	w.Write(blob)
}

func (h *VHost) builtinHandler(r *hrouter.Router) {
	r.GET(fmt.Sprintf("/%s/info", h.Name), h.httpInfo)
	r.GET(fmt.Sprintf("/%s/service/list", h.Name), h.httpListService)
	r.GET(fmt.Sprintf("/%s/impl/list", h.Name), h.httpListImpl)
}

// interface alog.HttpResponseSummary
func (r *responseWriterWrapper) Status() int {
	return r.status
}

// interface http.ResponseWriter
func (r *responseWriterWrapper) Header() http.Header {
	return r.w.Header()
}

func (r *responseWriterWrapper) Write(input []byte) (int, error) {
	return r.w.Write(input)
}

func (r *responseWriterWrapper) WriteHeader(status int) {
	r.status = status
	r.w.WriteHeader(status)
}

func (v *VHost) doLog(_ *alog.SessionLog) {
	// TODO(dpeng): Add log sinking services
}

func (v *VHost) doPhaseError(
	_ *vhostService,
	w http.ResponseWriter,
	_ *http.Request,
	_ hrouter.Params,
	err error,
	pindex int,
) {
	switch pindex {
	case phase.PhaseHttpAccess:
		w.WriteHeader(v.RejectStatus)
		w.Write([]byte(fmt.Sprintf("%s: %s", v.RejectBody, err.Error())))
		break

	default:
		w.WriteHeader(v.ErrorStatus)
		w.Write([]byte(fmt.Sprintf("%s: %s", v.ErrorBody, err.Error())))
		break
	}
}

// Session interface defined 3 stages for each module/services
func (v *VHost) doPhase(asvc *vhostService,
	w http.ResponseWriter, req *http.Request, param hrouter.Params) {

	handler, err := asvc.getSessionHandler()
	if err != nil {
		v.doPhaseError(asvc, w, req, param, err, phase.PhaseCreateSessionHandler)
		return
	}

	responseWrapper := &responseWriterWrapper{
		w: w,
	}

	log := &alog.SessionLog{
		Format:              v.LogFormat,
		HttpRequest:         req,
		RouterParams:        param,
		HttpResponseSummary: responseWrapper,
		VHost:               handler,
		Session:             handler,
	}

	defer func() {
		v.doLog(log)
		handler.finish()
	}()

	// 0. Run HPL setup
	{
		handler.setPhase(phase.PhaseInit, ".init")
		if err := handler.init(); err != nil {
			handler.errorDescription = err.Error()
			v.doPhaseError(asvc, w, req, param, err, phase.PhaseInit)
			return
		}
	}

	// 1. PhaseAccess
	{
		handler.setPhase(phase.PhaseHttpAccess, "http.access")
		if err := handler.Access(req, param); err != nil {
			handler.errorDescription = err.Error()
			v.doPhaseError(asvc, w, req, param, err, phase.PhaseHttpAccess)
			return
		}
	}

	// 2. PhaseRequest
	{
		handler.setPhase(phase.PhaseHttpRequest, "http.request")
		if err := handler.Request(req, param); err != nil {
			handler.errorDescription = err.Error()
			v.doPhaseError(asvc, w, req, param, err, phase.PhaseHttpRequest)
			return
		}
	}

	// 3. Run the session
	var sessionCtx interface{}
	{
		// 3.1) run session's start handler
		handler.setPhase(phase.PhaseSessionStart, "session.start")
		if err := handler.session.Start(handler); err != nil {
			handler.errorDescription = err.Error()
			v.doPhaseError(asvc, w, req, param, err, phase.PhaseSessionStart)
			return
		}
	}

	{
		// 3.2) run the session's prepare handler to create internal structure for
		//      using by the sessions
		handler.setPhase(phase.PhaseSessionPrepare, "session.Prepare")
		if ctx, err := handler.session.Prepare(req, param); err != nil {
			handler.errorDescription = err.Error()
			v.doPhaseError(asvc, w, req, param, err, phase.PhaseSessionPrepare)
			return
		} else {
			sessionCtx = ctx
		}
	}

	{
		// 3.3) run the session accept handler
		handler.setPhase(phase.PhaseSessionAccept, "session.Accept")
		if r, err := handler.session.Accept(sessionCtx); err != nil {
			handler.errorDescription = err.Error()
			v.doPhaseError(asvc, w, req, param, err, phase.PhaseSessionAccept)
			return
		} else {
			handler.sessionResult = r
		}
	}

	// 4. Generate the http response based on sessions result
	{
		handler.setPhase(phase.PhaseHttpResponse, "http.Response")
		if err := handler.Response(responseWrapper, req, param); err != nil {
			handler.errorDescription = err.Error()
			v.doPhaseError(asvc, w, req, param, err, phase.PhaseHttpResponse)
			return
		}
	}

	// Notes, we need to invoke session's done handler *after* the response been
	// generated since the session handler may have some data used by the response
	// handler
	{
		// 3.4) run the session's done handler
		handler.setPhase(phase.PhaseSessionDone, "session.Done")
		handler.session.Done(sessionCtx)
	}

	// 5. Lastly log generation
	{
		handler.setPhase(phase.PhaseAccessLog, ".AccessLog")
		handler.Log(log)
	}

}

func (h *VHost) Run() {
	h.HttpServer.ListenAndServe()
}

func newVHost(config *cfg.VHost) (*VHost, error) {
	vhost := &VHost{}
	router := hrouter.New()

	// misc
	vhost.ErrorStatus = cfg.NotZeroInt(
		vhost.ErrorStatus,
		g.VHostErrorStatus,
	)
	vhost.ErrorBody = cfg.NotZeroStr(
		vhost.ErrorBody,
		g.VHostErrorBody,
	)

	vhost.RejectStatus = cfg.NotZeroInt(
		vhost.RejectStatus,
		g.VHostRejectStatus,
	)
	vhost.RejectBody = cfg.NotZeroStr(
		vhost.RejectBody,
		g.VHostRejectBody,
	)

	{
		logFormat := cfg.NotZeroStr(
			config.LogFormat,
			g.VHostLogFormat,
		)

		logf, err := alog.NewSessionLogFormat(logFormat)
		if err != nil {
			return nil, err
		}
		vhost.LogFormat = logf
	}

	// ServiceList handling
	var serviceList []*vhostService

	for _, svc := range config.ServiceList {

		factory := service.GetServiceFactory(svc.Name)
		if factory == nil {
			return nil, fmt.Errorf("service %s is unknown", svc.Name)
		}
		service, err := factory.Create(svc)
		if err != nil {
			return nil, err
		}
		asvc, err := newvhostService(
			vhost,
			service,
			svc,
		)
		if err != nil {
			return nil, err
		}

		serviceList = append(serviceList, asvc)

		// router setup
		r := svc.Router
		methodList := svc.Method

		for _, method := range methodList {
			switch method {
			case "GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE":
				_, _, has := router.Lookup(method, r)
				if has {
					return nil, fmt.Errorf("service(%s), router(%s, %s) already existed",
						service.Name(), method, r)
				}
				router.Handle(method, r, func(w http.ResponseWriter, r *http.Request, p hrouter.Params) {
					vhost.doPhase(asvc, w, r, p)
				})
				break

			case "*":
				for _, m := range []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE"} {
					_, _, has := router.Lookup(m, r)
					if has {
						return nil, fmt.Errorf("service(%s) router(%s, %s) already existed",
							service.Name(), method, r)
					}
					router.Handle(method, r, func(w http.ResponseWriter, r *http.Request, p hrouter.Params) {
						vhost.doPhase(asvc, w, r, p)
					})
				}
				break

			default:
				return nil, fmt.Errorf("service(%s) unknown HTTP method %s",
					service.Name(), method)
			}
		}
	}

	// finish the creation of VHost object
	vhost.Name = config.Name
	vhost.Router = router
	vhost.ActiveService = serviceList
	vhost.HttpServer = &http.Server{
		Addr:           config.Endpoint,
		Handler:        router,
		ReadTimeout:    time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.WriteTimeout) * time.Second,
		MaxHeaderBytes: config.MaxHeaderSize,
	}
	vhost.clientPool = hclient.NewHClientPool(
		vhost.Name,
		cfg.NotZeroInt64(config.Resource.HttpClientPoolMaxSize, g.VHostHttpClientPoolMaxSize),
		cfg.NotZeroInt64(config.Resource.HttpClientTimeout, g.VHostHttpClientTimeout),
		cfg.NotZeroInt64(config.Resource.HttpClientMaxDrainSize, g.VHostHttpClientMaxDrainSize),
	)

	vhost.builtinHandler(router)

	return vhost, nil
}
