package module

// Concate services, ie generate a merge/concate of multiple http upstream
// content and send them as downstream response. This module allows highly
// flexible customization via policy engine.

// Each request will be splitted into 2 different phases. And the go code will
// drive each phase enter and leave

// 1. generate
//    In this phase, the policy engine's is responsible to generate a output
//    via action "url" which contains a list of strings that indicates the
//    URL a request must be made against to

// 2. check
//    In this phase, for each url in the lists, the request event will be
//    emitted and user is capable of fine grained customize the request by
//    setting its request's method, header, body, most importantly, a pass_status
//    action allow user to setup a failure condition when the request cannot be
//    made to downstream

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/phase"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/service"
)

const (
	concatePhaseUrl = iota
	concatePhaseRequest
	concatePhaseError
	concatePhaseResponse
)

const (
	concateRequestUrl = iota
	concateRequestRequest
	concateRequestString
	concateRequestStream
	concateRequestResponse
)

type concateServiceFactory struct{}

type concateService struct{}

type concateRequest struct {
	kind int

	// type RequestUrl
	url string

	// type RequestRequest
	request *http.Request

	// type RequestString
	dataString string

	// type stream
	dataStream io.ReadCloser

	// type response
	dataResponse *http.Response

	// script evaluation result
	rejectError   error
	pass          bool
	fetchResponse *http.Response
}

type concateSession struct {
	service *concateService
	r       service.SessionResource

	// background barrier for syncing the running goroutine that generates the write
	wg sync.WaitGroup

	// internal HPL used by the background go routine to evaluate |request| event
	// which cannot be the one used in the foreground go routine
	hpl *hpl.Hpl

	// context variable
	reqList []*concateRequest
	idx     int
	bgError error

	writer *io.PipeWriter
	reader *io.PipeReader

	bgWrapper hpl.SessionWrapper
}

func (s *concateService) Name() string {
	return "concate"
}

func (s *concateService) IDL() string {
	return ""
}

func (c *concateService) NewSession() (service.Session, error) {
	return &concateSession{
		service: c,
		r:       nil,
		hpl:     hpl.NewHpl(),
	}, nil
}

func (c *concateSession) resetForAccept() {
	c.reqList = nil
	c.idx = 0
	c.bgError = nil
	c.writer = nil
	c.reader = nil
	c.bgWrapper = nil
}

func (c *concateRequest) setRequest(req *http.Request) {
	c.request = req
	c.kind = concateRequestRequest
}

func (c *concateRequest) setUrl(url string) {
	c.url = url
	c.kind = concateRequestUrl
}

func (c *concateRequest) setString(data string) {
	c.dataString = data
	c.kind = concateRequestString
}

func (c *concateRequest) setStream(stream io.ReadCloser) {
	c.dataStream = stream
	c.kind = concateRequestStream
}

func (c *concateRequest) setResponse(resp *http.Response) {
	c.dataResponse = resp
	c.kind = concateRequestResponse
}

func (s *concateSession) curreq() *concateRequest {
	return s.reqList[s.idx]
}

func (s *concateSession) feedOneUrl(req *concateRequest) (*http.Response, error) {
	var hreq *http.Request

	if req.kind == concateRequestUrl {
		hh, err := http.NewRequest(
			"GET",
			req.url,
			strings.NewReader(""),
		)
		if err != nil {
			return nil, err
		}
		hreq = hh
	} else {
		hreq = req.request
	}

	client, err := s.bgWrapper.GetHttpClient(hreq.URL.String())
	if err != nil {
		return nil, err
	}

	return client.Do(hreq)
}

// the background goroutine used for evaluating the request. Notes this goroutine
// will be terminated once the whole session is done, ie inside of the Done call
func (s *concateSession) feeder() {
	closer := []io.Closer{}

	defer func() {
		for _, c := range closer {
			c.Close()
		}

		for i := s.idx; i < len(s.reqList); i++ {
			r := s.reqList[i]
			switch r.kind {

			case concateRequestStream:
				r.dataStream.Close()
				break

			case concateRequestResponse:
				r.dataResponse.Body.Close()
				break

			default:
				break
			}
		}

		// pipe been closed
		s.writer.Close()

		s.wg.Done()
	}()

	s.bgError = nil

	// background go routine that should be used at the back
LOOP:
	for idx, req := range s.reqList {

	AGAIN:
		s.idx = idx
		var output io.Reader
		var httpresp *http.Response

		// types of concate services needs to be performed
		switch req.kind {
		case concateRequestUrl, concateRequestRequest:
			// do the http request
			if hresp, err := s.feedOneUrl(req); err != nil {
				s.bgError = err
				break LOOP
			} else {
				httpresp = hresp
				output = hresp.Body
				closer = append(closer, hresp.Body)
			}
			break

		case concateRequestString:
			output = strings.NewReader(req.dataString)
			break

		case concateRequestStream:
			output = req.dataStream
			closer = append(closer, req.dataStream)
			break

		default:
			output = req.dataResponse.Body
			closer = append(closer, req.dataResponse.Body)
			break
		}

		// executing user's HPL event in case user want some modification, notes, the
		// trigger only happened when original request comming from http request
		if req.kind == concateRequestUrl || req.kind == concateRequestRequest {
			req.fetchResponse = httpresp
			if err := s.hpl.OnCustomize("check", s.bgWrapper); err != nil {
				s.bgError = err
				break LOOP
			}

			if req.rejectError != nil {
				s.bgError = req.rejectError
				break LOOP
			} else if !req.pass {
				// re-evaluate this case
				goto AGAIN
			}
		}

		_, err := io.Copy(s.writer, output)

		if err != nil {
			s.bgError = err
			break LOOP
		}
	}
}

func (s *concateSession) OnLoadVar(p int, _ *pl.Evaluator, name string) (pl.Val, error) {
	if p == phase.PhaseBackground {
		switch name {
		case "index":
			return pl.NewValInt(s.idx), nil

		case "response":
			return hpl.NewResponseVal(s.curreq().fetchResponse), nil

		default:
			break
		}
	}
	return pl.NewValNull(), fmt.Errorf("unknown variable %s", name)
}

func (s *concateSession) OnStoreVar(phase int, _ *pl.Evaluator, name string, _ pl.Val) error {
	return fmt.Errorf("unknown variable %s", name)
}

func (s *concateSession) OnCall(phase int, _ *pl.Evaluator, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("unknown function %s", name)
}

func (s *concateSession) onActionBackground(name string, val pl.Val) error {
	x := s.curreq()

	switch name {
	case "reject":
		if val.Type == pl.ValStr {
			x.pass = false
			x.rejectError = fmt.Errorf(val.String())
		} else {
			return fmt.Errorf("action 'reject''s argument must be string")
		}
		return nil

	case "pass":
		if val.Type == pl.ValBool {
			x.pass = val.Bool()
		} else {
			return fmt.Errorf("action 'pass''s argument must be bool")
		}
		return nil

	case "rewrite":
		switch val.Type {
		case pl.ValStr:
			x.setString(val.String())
			break

		default:
			if hpl.ValIsHttpBody(val) {
				body, _ := val.Usr().Context.(*hpl.Body)
				x.setStream(body.Stream().Stream)
			} else if hpl.ValIsReadableStream(val) {
				stream, _ := val.Usr().Context.(*hpl.ReadableStream)
				x.setStream(stream.Stream)
			} else if hpl.ValIsHttpResponse(val) {
				resp, _ := val.Usr().Context.(*hpl.Response)
				x.setResponse(resp.HttpResponse())
			} else {
				return fmt.Errorf("action 'rewrite''s argument must be string/stream/body/response")
			}
			break
		}
		return nil

	default:
		break
	}

	return fmt.Errorf("action %s is unknown", name)
}

func (s *concateSession) oneBackend(val pl.Val) (*concateRequest, error) {
	o := &concateRequest{}
	switch val.Type {
	case pl.ValStr:
		o.setString(val.String())
		break

	default:
		if hpl.ValIsUrl(val) {
			url, _ := val.Usr().Context.(*hpl.Url)
			o.setUrl(url.URL().String())
		} else if hpl.ValIsHttpRequest(val) {
			req, _ := val.Usr().Context.(*hpl.Request)
			o.setRequest(req.HttpRequest())
		} else if hpl.ValIsHttpBody(val) {
			body, _ := val.Usr().Context.(*hpl.Body)
			o.setStream(body.Stream().Stream)
		} else if hpl.ValIsReadableStream(val) {
			stream, _ := val.Usr().Context.(*hpl.ReadableStream)
			o.setStream(stream.Stream)
		} else if hpl.ValIsHttpResponse(val) {
			resp, _ := val.Usr().Context.(*hpl.Response)
			o.setResponse(resp.HttpResponse())
		} else {
			return nil, fmt.Errorf("invalid argument for 'output' action")
		}
		break
	}

	return o, nil
}

func (s *concateSession) onActionBackend(name string, val pl.Val) error {
	o := []*concateRequest{}

	switch name {
	case "output":
		if val.Type == pl.ValList {
			for _, v := range val.List().Data {
				c, err := s.oneBackend(v)
				if err != nil {
					return err
				}
				o = append(o, c)
			}
		} else {
			c, err := s.oneBackend(val)
			if err != nil {
				return err
			}
			o = append(o, c)
		}

		s.reqList = append(s.reqList, o...)
		return nil

	default:
		break
	}

	return fmt.Errorf("unknown action name %s", name)
}

func (s *concateSession) SessionResource() service.SessionResource {
	return s.r
}

func (s *concateSession) OnAction(p int, _ *pl.Evaluator, name string, val pl.Val) error {

	if p == phase.PhaseBackground {
		return s.onActionBackground(name, val)
	} else if p == phase.PhaseSessionAccept {
		return s.onActionBackend(name, val)
	}

	return fmt.Errorf("unknown function %s", name)
}

// service.Session

func (s *concateSession) Service() service.Service {
	return s.service
}

func (s *concateSession) Prepare(_ *http.Request, _ hrouter.Params) (interface{}, error) {
	return nil, nil
}

func (s *concateSession) Start(r service.SessionResource) error {
	s.r = r
	return nil
}

func (s *concateSession) hasPending() bool {
	return len(s.reqList) != 0
}

func (s *concateSession) Done(_ interface{}) {
	// wait for the background job to be done
	if s.hasPending() {
		s.wg.Wait()
	}

	if s.reader != nil {
		s.reader.Close()
	}
	if s.writer != nil {
		s.writer.Close()
	}
}

func (c *concateSession) Accept(_ interface{}, h *hpl.Hpl, wrapper hpl.SessionWrapper) (service.SessionResult, error) {
	c.resetForAccept()

	// create the pipe for background feeder to generate the output
	{
		r, w := io.Pipe()
		c.reader = r
		c.writer = w
	}

	// create a backgronud evaluator's session wrapper
	c.bgWrapper = newBgSessionWrapper(
		wrapper,
		c,
	)

	// run hpl
	err := h.OnCustomize(
		"generate",
		wrapper,
	)

	if err != nil {
		return service.SessionResult{}, err
	}

	c.hpl.SetPolicy(h.Policy)

	if c.hasPending() {
		c.wg.Add(1)

		// running the feeder in background to generate the response
		go c.feeder()
	} else {
		c.writer.Close()
	}

	return service.SessionResult{
		Event: "response",
		Vars: []pl.DynamicVariable{
			pl.DynamicVariable{
				Key:   "output",
				Value: hpl.NewReadableStreamValFromStream(c.reader),
			},
		},
	}, nil
}

func (c *concateServiceFactory) Name() string {
	return "concate"
}

func (c *concateServiceFactory) IDL() string {
	return ""
}

func (c *concateServiceFactory) Comment() string {
	return `
A service that can combine/concate multiple http upstream into a single response
Typically used for combining assets load from upstream
`
}

func (c *concateServiceFactory) Create(_ *config.Service) (service.Service, error) {
	return &concateService{}, nil
}

func init() {
	service.RegisterServiceFactory(&concateServiceFactory{})
}
