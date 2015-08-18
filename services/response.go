// Response module helps to construct idiomatic
// responses to http request
package services

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type ErrorContent struct {
	Type string `json:"type,omitempty"`
	Status int `json:"status"`
	Message string `json:"message"`
	Param string `json:"param,omitempty"`
}

type APIError struct {
	Error ErrorContent `json:"error"`
}

type Response struct {
	res http.ResponseWriter
	statusCode int
	Param string
}

// set param
func (resp *Response) ErrParam(param string) *Response {
	resp.Param = param
	return resp
}

// set status code
func (resp *Response) StatusCode(code int) *Response {
	resp.statusCode = code
	return resp
}

// sets status code to 200
func (resp *Response) OK() *Response {
	resp.statusCode = 200
	return resp
}

// sets status code to 404
func (resp *Response) NotFound() *Response {
	resp.statusCode = 404
	return resp
}

func (resp *Response) SetHeader(key, val string) *Response {
	resp.res.Header().Set(key, val)
	return resp
}

// write/send a string or and object data as the response
func (resp *Response) Send(body interface{}) (int, error) {
	resp.res.WriteHeader(resp.statusCode)
	switch o := body.(type) {
	case string:
		return fmt.Fprint(resp.res, o)
	default:
		return fmt.Fprint(resp.res, fmt.Sprintf("%#v", o))
	}
	
}

// write/send json response
func (resp *Response) Json(obj interface{}) (int, error) {
	if jsonData, err := json.Marshal(obj); err == nil {
		resp.SetHeader("Content-Type", "application/json")
		return fmt.Fprint(resp.res, string(jsonData))
	} else {
		fmt.Println(err, obj)
		return fmt.Fprint(resp.res, "error generating response")
	}
}

// return an api error
func (resp *Response) Error(status int, msgType string, msg string) (int, error) {
	e := APIError {
		Error: ErrorContent {
			Type: msgType,
			Status: status,
			Message: msg,
			Param: resp.Param,
		},
	}
	return resp.Json(e)
}


// create a new response object
func Res(res http.ResponseWriter) *Response {
	return &Response{ res: res, statusCode: 200 }
}
