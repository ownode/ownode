package middlewares

import (
	"net/http"
	"github.com/ownode/config"
	"github.com/go-martini/martini"
	"strings"
	"github.com/ownode/services"
	"regexp"
)

type MiddlewareFunc func(martini.Context, http.ResponseWriter, *http.Request, *config.CustomLog)
type PolicyFunc func(http.ResponseWriter, services.AuxRequestContext, *config.CustomLog)

type policy struct {
	policies map[string][]PolicyFunc
	req services.AuxRequestContext
	res http.ResponseWriter
	log *config.CustomLog
}

// match policy path to the a request url path
// policy path can be full path and can have wildcard `*`
func (pol *policy) IsMatch(policyPath, requestPath string, reqMethod string) bool {

	// determine request method of policy path
	method := "get"
	if services.StringStartsWith(strings.ToLower(policyPath), "post") {
		method = "post"
	}

	// ensure policy path request method matches the actual request method
	if method != strings.ToLower(reqMethod) {
		services.Println("matches not ", method, reqMethod)
		return false
	}

	// reassign policy path to the second substr of policyPath passesed in 
	// if it contains a request method declaration
	policyPathSplit := services.StringSplitBySpace(policyPath)
	if len(policyPathSplit) > 1 {
		policyPath = policyPathSplit[1]
	}

	// change any wildcard to proper regex repeating operator `.*`
	policyPath = strings.Replace(policyPath, "*", ".*", -1)

	// check if policy path matches request path
	matched, err := regexp.MatchString(policyPath, requestPath)
	if err != nil {
		panic(err)
	}

	return matched
}

// call all policies for the current request.
// policies for the current matching URL path is called
func (pol *policy) Process() {
	for path, funcList := range pol.policies {
		if pol.IsMatch(path, pol.req.URL.Path, pol.req.Method) {
			for _, f := range funcList {
				if pol.req.Written() == false {
					f(pol.res, pol.req, pol.log)
				}
			}
		}
	}
}

func Policies(policies map[string][]PolicyFunc) interface{} {
	return func(c martini.Context, res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog){
		pol := policy{ policies, req, res, log }
		pol.Process()
	}
}