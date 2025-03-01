package endpoints

import (
	"encoding/json"
	"net/http"
)

var (
	HeaderContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	ApplicationJSON = "application/json"
	TextHTML        = "text/html"
)

type Req struct {
	w http.ResponseWriter
	r *http.Request
}

func (r *Req) WantsJSON() bool {
	return r.r.Header.Get(HeaderContentType) == ApplicationJSON
}

func (r *Req) WriteJSON(statusCode int, content interface{}) {
	r.w.Header().Set(HeaderContentType, ApplicationJSON)
	r.w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(r.w).Encode(content)
	if err != nil {
		panic(err)
	}
}
