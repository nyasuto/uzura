package js

import (
	"github.com/nyasuto/uzura/internal/dom"
)

// ExecuteScripts finds and executes all inline <script> tags in document order.
// Deferred scripts run after all non-deferred scripts.
// Errors in individual scripts do not stop execution of subsequent scripts.
// Returns a slice of errors from scripts that threw.
func ExecuteScripts(vm *VM, doc *dom.Document) []error {
	scripts := doc.GetElementsByTagName("script")

	var inline []*dom.Element
	var deferred []*dom.Element

	for _, s := range scripts {
		if s.HasAttribute("src") {
			continue
		}
		if s.HasAttribute("defer") {
			deferred = append(deferred, s)
		} else {
			inline = append(inline, s)
		}
	}

	var errs []error

	for _, s := range inline {
		if err := execScript(vm, s); err != nil {
			errs = append(errs, err)
		}
	}

	for _, s := range deferred {
		if err := execScript(vm, s); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func execScript(vm *VM, s *dom.Element) error {
	code := s.TextContent()
	if code == "" {
		return nil
	}
	_, err := vm.Eval(code)
	return err
}
