package cdp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
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

// Register adds Runtime domain handlers to the registry.
func (d *RuntimeDomain) Register(r HandlerRegistry) {
	r.HandleSession("Runtime.enable", d.enable)
	r.HandleSession("Runtime.evaluate", d.evaluate)
	r.HandleSession("Runtime.callFunctionOn", d.callFunctionOn)
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
	raw, err := vm.EvalRaw(p.Expression)

	var events []Event

	if err != nil {
		exObj := d.objects.SerializeValue(err.Error())
		exDetails := d.buildExceptionDetails(err.Error(), p.Expression)
		result := map[string]interface{}{
			"result":           exObj,
			"exceptionDetails": exDetails,
		}
		evt := d.makeExceptionThrownEvent(exDetails)
		if evt != nil {
			events = append(events, *evt)
		}
		r, merr := json.Marshal(result)
		return r, events, merr
	}

	ro := d.objects.SerializeGojaValue(raw)
	if p.ReturnByValue && ro.ObjectID != "" {
		// When returnByValue is true, export and inline the value.
		ro = d.objects.SerializeValue(raw.Export())
	}
	result := map[string]interface{}{"result": ro}
	r, merr := json.Marshal(result)
	return r, events, merr
}

func (d *RuntimeDomain) callFunctionOn(_ *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		FunctionDeclaration string `json:"functionDeclaration"`
		ObjectID            string `json:"objectId"`
		ExecutionContextID  int    `json:"executionContextId"`
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

	// Resolve this object — keep as goja.Value if possible.
	thisGojaVal := goja.Undefined()
	if p.ObjectID != "" {
		stored, ok := d.objects.Get(p.ObjectID)
		if !ok {
			return nil, nil, fmt.Errorf("object not found: %s", p.ObjectID)
		}
		if gv, ok := stored.(goja.Value); ok {
			thisGojaVal = gv
		} else {
			thisGojaVal = rt.ToValue(stored)
		}
	}

	// Build argument goja values.
	gojaArgs := make([]goja.Value, len(p.Arguments))
	for i, arg := range p.Arguments {
		if arg.ObjectID != "" {
			stored, ok := d.objects.Get(arg.ObjectID)
			if !ok {
				return nil, nil, fmt.Errorf("argument object not found: %s", arg.ObjectID)
			}
			if gv, ok := stored.(goja.Value); ok {
				gojaArgs[i] = gv
			} else {
				gojaArgs[i] = rt.ToValue(stored)
			}
		} else {
			gojaArgs[i] = rt.ToValue(arg.Value)
		}
	}

	// Set up temporaries for the call expression.
	_ = rt.Set("__cdp_this", thisGojaVal)
	for i, arg := range gojaArgs {
		_ = rt.Set(fmt.Sprintf("__cdp_arg%d", i), arg)
	}

	callExpr := fmt.Sprintf("(%s).call(__cdp_this", p.FunctionDeclaration)
	for i := range gojaArgs {
		callExpr += fmt.Sprintf(", __cdp_arg%d", i)
	}
	callExpr += ")"

	raw, err := vm.EvalRaw(callExpr)

	// Clean up temp globals.
	_ = rt.Set("__cdp_this", goja.Undefined())
	for i := range gojaArgs {
		_ = rt.Set(fmt.Sprintf("__cdp_arg%d", i), goja.Undefined())
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

	ro := d.objects.SerializeGojaValue(raw)
	if p.ReturnByValue && ro.ObjectID != "" {
		ro = d.objects.SerializeValue(raw.Export())
	}
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
