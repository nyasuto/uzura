package js

import (
	"github.com/dop251/goja"
	"github.com/nyasuto/uzura/internal/dom"
)

// datasetProxy implements goja.DynamicObject for element.dataset Proxy access.
type datasetProxy struct {
	ds *dom.Dataset
	rt *goja.Runtime
}

func (d *datasetProxy) Get(key string) goja.Value {
	if !d.ds.Has(key) {
		return goja.Undefined()
	}
	return d.rt.ToValue(d.ds.Get(key))
}

func (d *datasetProxy) Set(key string, val goja.Value) bool {
	d.ds.Set(key, val.String())
	return true
}

func (d *datasetProxy) Has(key string) bool {
	return d.ds.Has(key)
}

func (d *datasetProxy) Delete(key string) bool {
	d.ds.Delete(key)
	return true
}

func (d *datasetProxy) Keys() []string {
	all := d.ds.All()
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	return keys
}
