package service

import (
	"github.com/dianpeng/mono-service/alog"
	hrouter "github.com/julienschmidt/httprouter"
	"net/http"
	"sync"
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// interface for getting various session related resource for internal usage
type SessionResource interface {
	// get a http client object from the underly resource pool
	GetHttpClient(url string) (HttpClient, error)
}

// entry for handling a single http request/response
type Session interface {
	// --------------------------------------------------------------------------
	// module phase handler

	// Prepare phase, ie any session internal initialization can be placed here,
	// like running the HPL global constructor etc ...
	Start(SessionResource) error

	// Invoked when the http request is just received, ie checking whther we are
	// allowed to use the request
	Allow(*http.Request, hrouter.Params) error

	// Invoked when the http request is been accepted by the session handler
	Accept(http.ResponseWriter, *http.Request, hrouter.Params)

	// Invoked after the Accept is done and the server wants to generate access log
	Log(*alog.SessionLog) error

	// Session is done
	Done()

	// --------------------------------------------------------------------------
	// Get the last start binded SessionResource object, if not Start after Done,
	// then this function can return nil
	SessionResource() SessionResource

	// Return the service belonged to the Session
	Service() Service
}

// SessionList implementation, help upper controller to manage different session
// object when facing various incomming requests
type SessionList struct {
	Idle    []Session
	MaxSize int
	sync.Mutex
}

func (s *SessionList) IdleSize() int {
	return len(s.Idle)
}

func (s *SessionList) Get() Session {
	s.Lock()
	defer s.Unlock()

	if len(s.Idle) == 0 {
		return nil
	}
	idleSize := len(s.Idle)
	last := s.Idle[idleSize-1]
	s.Idle = s.Idle[:idleSize-1]
	return last
}

func (s *SessionList) Put(session Session) bool {
	s.Lock()
	defer s.Unlock()

	if len(s.Idle)+1 > s.MaxSize {
		return false
	}
	s.Idle = append(s.Idle, session)
	return true
}
