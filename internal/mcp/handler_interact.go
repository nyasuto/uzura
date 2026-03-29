package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// InteractParams represents the arguments for the interact tool.
type InteractParams struct {
	URL      string `json:"url"`
	Selector string `json:"selector"`
	Action   string `json:"action"`
	Value    string `json:"value,omitempty"`
}

// InteractTool returns the tool definition for the interact tool.
func InteractTool() Tool {
	return Tool{
		Name:        "interact",
		Description: "ページ上の要素をクリックまたはフォーム入力する",
		InputSchema: json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "対象ページのURL"
		},
		"selector": {
			"type": "string",
			"description": "操作対象のCSSセレクター"
		},
		"action": {
			"type": "string",
			"enum": ["click", "fill"],
			"description": "実行するアクション"
		},
		"value": {
			"type": "string",
			"description": "fill時の入力値"
		}
	},
	"required": ["url", "selector", "action"]
}`),
	}
}

// RegisterInteractTool registers the interact tool with its handler on the server.
func RegisterInteractTool(s *Server) {
	s.Tools.RegisterWithHandler(InteractTool(), func(arguments json.RawMessage) (*ToolCallResult, error) {
		return handleInteract(s.Session, arguments)
	})
}

func handleInteract(session *PageSession, arguments json.RawMessage) (*ToolCallResult, error) {
	var params InteractParams
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if params.URL == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "url is required"}
	}
	if params.Selector == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "selector is required"}
	}
	if params.Action != "click" && params.Action != "fill" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "action must be 'click' or 'fill'"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), browseTimeout)
	defer cancel()

	p, err := session.GetOrNavigate(ctx, params.URL)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("error: %s", err)}},
			IsError: true,
		}, nil
	}

	doc := p.Document()
	if doc == nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "error: no document loaded"}},
			IsError: true,
		}, nil
	}

	el, err := doc.DocumentElement().QuerySelector(params.Selector)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("selector error: %s", err)}},
			IsError: true,
		}, nil
	}
	if el == nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("no element matches selector: %s", params.Selector)}},
			IsError: true,
		}, nil
	}

	vm := p.VM()

	switch params.Action {
	case "click":
		script := fmt.Sprintf(`(function() {
			var el = document.querySelector(%q);
			if (!el) return "element not found";
			var evt = new Event("click", {bubbles: true, cancelable: true});
			el.dispatchEvent(evt);
			return "clicked";
		})()`, params.Selector)
		val, evalErr := vm.Eval(script)
		if evalErr != nil {
			return &ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("click error: %s", evalErr)}},
				IsError: true,
			}, nil
		}
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("%v", val)}},
		}, nil

	case "fill":
		script := fmt.Sprintf(`(function() {
			var el = document.querySelector(%q);
			if (!el) return "element not found";
			el.value = %q;
			var inputEvt = new Event("input", {bubbles: true});
			el.dispatchEvent(inputEvt);
			var changeEvt = new Event("change", {bubbles: true});
			el.dispatchEvent(changeEvt);
			return "filled";
		})()`, params.Selector, params.Value)
		val, evalErr := vm.Eval(script)
		if evalErr != nil {
			return &ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("fill error: %s", evalErr)}},
				IsError: true,
			}, nil
		}
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("%v", val)}},
		}, nil
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: "unknown action"}},
		IsError: true,
	}, nil
}
