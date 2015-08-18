// Auxilliary request context as a service.
// Wraps martini.Request and martini.Context and also provides
// data storage for context sharing accross middlewares
package services

import (
	"net/http"
	"github.com/go-martini/martini"
)

type AuxRequestContext struct {
	martini.Context
	*http.Request
	data map[string]interface{}
}

// add data
func (rc *AuxRequestContext) SetData(key string, val interface{}){
	rc.data[key] = val
}

// get data
func (rc *AuxRequestContext) GetData(key string) interface{} {
	return rc.data[key]
}

// return new customer request writer
func NewAuxRequestContext(c martini.Context, rw *http.Request) AuxRequestContext {
	return AuxRequestContext{ c, rw, make(map[string]interface{}) }
}
