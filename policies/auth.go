package policies

import (
	"net/http"
	"github.com/ownode/config"
	"github.com/ownode/services"
	"strings"
)

// ensures current request has an `Authorization` header
func MustHaveAuthHeader(res http.ResponseWriter, arc services.AuxRequestContext, log *config.CustomLog) {
	if arc.Header.Get("Authorization") == "" {
		services.Res(res).Error(401, "invalid_request",  "missing authorization header field")
	}
}

// ensures authorization header is a `Basic` scheme
func MustBeBasic(res http.ResponseWriter, arc services.AuxRequestContext, log *config.CustomLog) {
	authorization := strings.ToLower(arc.Header.Get("Authorization"))
	if !services.StringStartsWith(authorization, "basic") {
		services.Res(res).Error(401, "invalid_request",  "authorization scheme must be Basic")
	}
}

// ensures authorizaion header is a `Bearer` scheme
func MustBeBearer(res http.ResponseWriter, arc services.AuxRequestContext, log *config.CustomLog) {
	authorization := strings.ToLower(arc.Header.Get("Authorization"))
	if !services.StringStartsWith(authorization, "bearer") {
		services.Res(res).Error(401, "invalid_request",  "authorization scheme must be Bearer")
	}
}

