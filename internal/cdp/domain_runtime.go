package cdp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/page"
)

// RuntimeDomain handles CDP Runtime domain methods.
type RuntimeDomain struct {
	page    *page.Page
	objects *ObjectStore
	session *Session // set on enable for async events
}

// NewRuntimeDomain creates a RuntimeDomain wrapping the given page.
// If p is nil, call SetPage before use.
func NewRuntimeDomain(p *page.Page) *RuntimeDomain {
	return &RuntimeDomain{
		page:    p,
		objects: NewObjectStore(),
	}
}

// SetPage sets the page for this domain. Used when the page must be created
// after the domain (e.g. to wire up ConsoleCallback).
func (d *RuntimeDomain) SetPage(p *page.Page) {
	d.page = p
}

// Register adds Runtime domain handlers to the server.
func (d *RuntimeDomain) Register(s *Server) {
	s.HandleSession("Runtime.enable", d.enable)
	s.HandleSession("Runtime.evaluate", d.evaluate)
	s.HandleSession("Runtime.callFunctionOn", d.callFunctionOn)
}

func (d *RuntimeDomain) enable(sess *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	d.session = sess
	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *RuntimeDomain) evaluate(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		Expression        string `json:"expression"`
		ReturnByValue     bool   `json:"returnByValue"`
		AwaitPromise      bool   `json:"awaitPromise"`
		GeneratePreview   bool   `json:"generatePreview"`
		IncludeCommandAPI bool   `json:"includeCommandLineAPI"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	vm := d.page.VM()
	val, err := vm.Eval(p.Expression)

	var events []Event

	if err != nil {
		// Build exception details for JS errors.
		exObj := d.objects.SerializeValue(err.Error())
		exDetails := d.buildExceptionDetails(err.Error(), p.Expression)
		result := map[string]interface{}{
			"result":           exObj,
			"exceptionDetails": exDetails,
		}

		// Also emit Runtime.exceptionThrown event.
		evt := d.makeExceptionThrownEvent(exDetails)
		if evt != nil {
			events = append(events, *evt)
		}

		r, merr := json.Marshal(result)
		return r, events, merr
	}

	ro := d.objects.SerializeValue(val)
	result := map[string]interface{}{"result": ro}
	r, merr := json.Marshal(result)
	return r, events, merr
}

func (d *RuntimeDomain) callFunctionOn(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		FunctionDeclaration string `json:"functionDeclaration"`
		ObjectID            string `json:"objectId"`
		Arguments           []struct {
			Value    interface{} `json:"value,omitempty"`
			ObjectID string      `json:"objectId,omitempty"`
		} `json:"arguments"`
		ReturnByValue bool `json:"returnByValue"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	vm := d.page.VM()
	rt := vm.Runtime()

	// Resolve this object.
	var thisVal interface{}
	if p.ObjectID != "" {
		var ok bool
		thisVal, ok = d.objects.Get(p.ObjectID)
		if !ok {
			return nil, nil, fmt.Errorf("object not found: %s", p.ObjectID)
		}
	}

	// Build argument values for the call.
	args := make([]interface{}, len(p.Arguments))
	for i, arg := range p.Arguments {
		if arg.ObjectID != "" {
			v, ok := d.objects.Get(arg.ObjectID)
			if !ok {
				return nil, nil, fmt.Errorf("argument object not found: %s", arg.ObjectID)
			}
			args[i] = v
		} else {
			args[i] = arg.Value
		}
	}

	// Evaluate: wrap as IIFE with this binding.
	// (function(){...}).call(thisArg, arg0, arg1, ...)
	_ = rt.Set("__cdp_this", thisVal)
	for i, arg := range args {
		_ = rt.Set(fmt.Sprintf("__cdp_arg%d", i), arg)
	}

	callExpr := fmt.Sprintf("(%s).call(__cdp_this", p.FunctionDeclaration)
	for i := range args {
		callExpr += fmt.Sprintf(", __cdp_arg%d", i)
	}
	callExpr += ")"

	val, err := vm.Eval(callExpr)

	// Clean up temp globals.
	_ = rt.Set("__cdp_this", nil)
	for i := range args {
		_ = rt.Set(fmt.Sprintf("__cdp_arg%d", i), nil)
	}

	if err != nil {
		exObj := d.objects.SerializeValue(err.Error())
		exDetails := d.buildExceptionDetails(err.Error(), p.FunctionDeclaration)
		result := map[string]interface{}{
			"result":           exObj,
			"exceptionDetails": exDetails,
		}
		r, merr := json.Marshal(result)
		return r, nil, merr
	}

	ro := d.objects.SerializeValue(val)
	result := map[string]interface{}{"result": ro}
	r, merr := json.Marshal(result)
	return r, nil, merr
}

// ConsoleCallback returns a js.ConsoleCallback that emits Runtime.consoleAPICalled events.
func (d *RuntimeDomain) ConsoleCallback() js.ConsoleCallback {
	return func(method string, args []interface{}) {
		if d.session == nil {
			return
		}
		remoteArgs := d.objects.SerializeForConsole(args)
		evt := map[string]interface{}{
			"type":               method,
			"args":               remoteArgs,
			"executionContextId": 1,
			"timestamp":          float64(time.Now().UnixMilli()) / 1000.0,
		}
		_ = d.session.SendEvent("Runtime.consoleAPICalled", evt)
	}
}

func (d *RuntimeDomain) buildExceptionDetails(errMsg, text string) map[string]interface{} {
	return map[string]interface{}{
		"exceptionId":  1,
		"text":         "Uncaught",
		"lineNumber":   0,
		"columnNumber": 0,
		"exception": RemoteObject{
			Type:        "object",
			Subtype:     "error",
			ClassName:   "Error",
			Description: errMsg,
		},
	}
}

func (d *RuntimeDomain) makeExceptionThrownEvent(details map[string]interface{}) *Event {
	ts := float64(time.Now().UnixMilli()) / 1000.0
	evtData, err := json.Marshal(map[string]interface{}{
		"timestamp":        ts,
		"exceptionDetails": details,
	})
	if err != nil {
		return nil
	}
	return &Event{Method: "Runtime.exceptionThrown", Params: evtData}
}
