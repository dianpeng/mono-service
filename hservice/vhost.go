package hservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dianpeng/mono-service/alog"
	cfg "github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/g"
	"github.com/dianpeng/mono-service/hclient"
	_ "github.com/dianpeng/mono-service/impl"
	"github.com/dianpeng/mono-service/service"
	"github.com/dianpeng/mono-service/util"
	hrouter "github.com/julienschmidt/httprouter"
)

type aService struct {
	svc         service.Service
	sessionPool service.SessionList
}

type VHost struct {
	Name          string
	ActiveService []*aService
	Router        *hrouter.Router
	HttpServer    *http.Server
	LogFormat     *alog.SessionLogFormat
	ErrorStatus   int
	ErrorBody     string
	RejectStatus  int
	RejectBody    string

	clientPool *hclient.HClientPool
}

func (h *VHost) Run() {
	h.HttpServer.ListenAndServe()
}

// builtin handlers
func (h *VHost) httpListService(w http.ResponseWriter, _ *http.Request, _ hrouter.Params) {
	var o []interface{}

	for _, svc := range h.ActiveService {
		o = append(o, map[string]interface{}{
			"name":           svc.svc.Name(),
			"idl":            svc.svc.IDL(),
			"policy":         util.JSONSplitLine(svc.svc.Policy()),
			"policyDump":     util.JSONSplitLine(svc.svc.PolicyDump()),
			"router":         svc.svc.Router(),
			"methodList":     svc.svc.MethodList(),
			"idleSession":    svc.sessionPool.IdleSize(),
			"maxSessionSize": svc.sessionPool.MaxSize,
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
	o["service#"] = len(h.ActiveService)
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

// we do a simple wrapper around http.ResponseWriter to adapt to our framework
type responseWriterWrapper struct {
	w      http.ResponseWriter
	status int
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

// interface alog.HttpResponseSummary
func (r *responseWriterWrapper) Status() int {
	return r.status
}

// interface for alog.VHostInformation
func (v *VHost) VHostName() string {
	return v.Name
}

func (v *VHost) VHostEndpoint() string {
	return v.HttpServer.Addr
}

type sessionInfoWrapper struct {
	session          service.Session
	phase            string
	errorDescription string
}

// interface for alog.SessionInfo
func (s *sessionInfoWrapper) ServiceName() string {
	return s.session.Service().Name()
}

func (s *sessionInfoWrapper) SessionPhase() string {
	return s.phase
}

func (s *sessionInfoWrapper) ErrorDescription() string {
	return s.errorDescription
}

func (v *VHost) doLog(_ *alog.SessionLog) {
	// TODO(dpeng): Add log sinking services
}

type sRes struct {
	vhost *VHost

	// list of client that is been used by this session
	activeHttpClient []*hclient.HClient
}

func (s *sRes) GetHttpClient(url string) (service.HttpClient, error) {
	c, err := s.vhost.clientPool.Get(url)
	if err != nil {
		return nil, err
	}
	s.activeHttpClient = append(s.activeHttpClient, &c)
	return &c, nil
}

func (s *sRes) finish() {
	// return all the active http client used by the current session
	if s.activeHttpClient != nil {
		for _, c := range s.activeHttpClient {
			s.vhost.clientPool.Put(*c)
		}
	}
}

// Session interface defined 3 stages for each module/services
func (v *VHost) doHttp(asvc *aService,
	w http.ResponseWriter, req *http.Request, param hrouter.Params) {

	// (0) preparation of a http request's handling
	session := asvc.sessionPool.Get()
	if session == nil {
		s, err := asvc.svc.NewSession()
		if err != nil {
			util.ErrorRequest(w, err)
			return
		}
		session = s
	}

	responseWrapper := &responseWriterWrapper{
		w: w,
	}

	sessionWrapper := &sessionInfoWrapper{
		session: session,
	}

	log := &alog.SessionLog{
		Format:              v.LogFormat,
		HttpRequest:         req,
		RouterParams:        param,
		HttpResponseSummary: responseWrapper,
		VHost:               v,
		Session:             sessionWrapper,
	}

	res := &sRes{
		vhost: v,
	}

	defer func() {
		asvc.sessionPool.Put(session)
		v.doLog(log)
		res.finish()
	}()

	// 1 Run moudle's phase handler to generate correct result
	{
		// 1.1) run allow function to check whether the incomming request is allowed
		//      on this module or not. This gives a simple way to guard unwantted
		//      request
		sessionWrapper.phase = "start"
		if err := session.Start(res); err != nil {
			sessionWrapper.errorDescription = err.Error()
			responseWrapper.WriteHeader(v.ErrorStatus)
			responseWrapper.Write([]byte(v.ErrorBody))
			return
		}
	}

	{
		// 1.2) run allow function to check whether the incomming request is allowed
		//      on this module or not. This gives a simple way to guard unwantted
		//      request
		sessionWrapper.phase = "allow"
		if err := session.Allow(req, param); err != nil {
			sessionWrapper.errorDescription = err.Error()
			responseWrapper.WriteHeader(v.RejectStatus)
			responseWrapper.Write([]byte(v.RejectBody))
			return
		}
	}

	{
		// 1.3) run the accept phase handler to generate module's output
		sessionWrapper.phase = "accept"
		session.Accept(responseWrapper, req, param)
	}

	{
		// 1.4) run the log phase handler to generate module's log output
		sessionWrapper.phase = "log"
		_ = session.Log(log)
	}

	{
		// 1.5) last finish the handler
		sessionWrapper.phase = "done"
		session.Done()
	}
}

func newSessionList(config *cfg.Service) service.SessionList {
	cache_size := config.MaxSessionCacheSize
	if cache_size == 0 {
		cache_size = g.MaxSessionCacheSize
	}
	return service.SessionList{
		MaxSize: cache_size,
	}
}

func newVHost(config *cfg.VHost) (*VHost, error) {
	vhost := &VHost{}
	router := hrouter.New()

	// misc
	if config.ErrorStatus == 0 {
		vhost.ErrorStatus = g.VHostErrorStatus
	} else {
		vhost.ErrorStatus = config.ErrorStatus
	}

	if config.ErrorBody == "" {
		vhost.ErrorBody = g.VHostErrorBody
	} else {
		vhost.ErrorBody = config.ErrorBody
	}

	if config.RejectStatus == 0 {
		vhost.RejectStatus = g.VHostRejectStatus
	} else {
		vhost.RejectStatus = config.RejectStatus
	}

	if config.RejectBody == "" {
		vhost.RejectBody = g.VHostRejectBody
	} else {
		vhost.RejectBody = config.RejectBody
	}

	logFormat := config.LogFormat
	if logFormat == "" {
		logFormat = g.VHostLogFormat
	}

	logf, err := alog.NewSessionLogFormat(logFormat)
	if err != nil {
		return nil, err
	}
	vhost.LogFormat = logf

	// ServiceList handling
	var serviceList []*aService

	for _, svc := range config.ServiceList {

		factory := service.GetServiceFactory(svc.Name)
		if factory == nil {
			return nil, fmt.Errorf("service %s is unknown", svc.Name)
		}
		service, err := factory.Create(svc)
		if err != nil {
			return nil, err
		}

		asvc := &aService{
			svc:         service,
			sessionPool: newSessionList(svc),
		}

		serviceList = append(serviceList, asvc)

		// based on service's Router and MethodList to decide how to route the
		// traffic to the router
		r := service.Router()
		methodList := service.MethodList()

		for _, method := range methodList {
			switch method {
			case "GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE":
				_, _, has := router.Lookup(method, r)
				if has {
					return nil, fmt.Errorf("service(%s), router(%s, %s) already existed",
						service.Name(), method, r)
				}
				router.Handle(method, r, func(w http.ResponseWriter, r *http.Request, p hrouter.Params) {
					vhost.doHttp(asvc, w, r, p)
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
						vhost.doHttp(asvc, w, r, p)
					})
				}
				break

			default:
				return nil, fmt.Errorf("service(%s) unknown HTTP method %s",
					service.Name(), method)
			}
		}
	}

	// (1) create the VHost object
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
