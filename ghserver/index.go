package ghserver

import (
	"github.com/exlinc/golang-utils/jsonhttp"
	"net/http"
)

func index(w http.ResponseWriter, r *http.Request) {
	// TODO check service health
	jsonhttp.JSONSuccess(w, nil, "Server healthy")
}
