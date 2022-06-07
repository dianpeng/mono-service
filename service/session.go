package service

import (
	"github.com/dianpeng/mono-service/pl"
	hrouter "github.com/julienschmidt/httprouter"
	"net/http"
)

// interface for getting various session related resource for internal usage
type SessionResource interface{}

// when a session finish its execution, it returns back a SessionResult object
// for exposing back to the hpl environment
type SessionResult struct {
	Event string
	Vars  []pl.DynamicVariable
}

// entry for handling a single http request/response
type Session interface {
	// --------------------------------------------------------------------------
	// module phase handler

	// Prepare phase, ie any session internal initialization can be placed here,
	// like running the HPL global constructor etc ...
	Start(SessionResource) error

	// Prepare a transparent request object to be used by the session's Accept
	Prepare(*http.Request, hrouter.Params) (interface{}, error)

	// Invoked when the http request is been accepted by the session handler
	Accept(context interface{}) (SessionResult, error)

	// The service session is terminated
	Done(context interface{})

	// --------------------------------------------------------------------------
	// HPL context usage
	OnLoadVar(int, *pl.Evaluator, string) (pl.Val, error)
	OnStoreVar(int, *pl.Evaluator, string, pl.Val) error
	OnCall(int, *pl.Evaluator, string, []pl.Val) (pl.Val, error)
	OnAction(int, *pl.Evaluator, string, pl.Val) error

	// --------------------------------------------------------------------------
	// Get the last start binded SessionResource object, if not Start after Done,
	// then this function can return nil
	SessionResource() SessionResource

	// Return the service belonged to the Session
	Service() Service
}
