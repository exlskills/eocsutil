package ghserver

import (
	"net/http"
)
import "github.com/exlinc/golang-utils/jsonhttp"

// `Use` allows us to stack middleware to process the request
// Example taken from https://github.com/gorilla/mux/pull/36#issuecomment-25849172
func Use(handler http.HandlerFunc, mid ...func(http.Handler) http.HandlerFunc) http.HandlerFunc {
	for _, m := range mid {
		handler = m(handler)
	}
	return handler
}

func RecoverAndLog(handler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			r := recover()
			if r != nil {
				jsonhttp.JSONInternalError(w, "An internal server error occurred", "Please try again in a few seconds")
				Log.Error("Panic occurred in HTTP handler:", r)
			}
		}()
		handler.ServeHTTP(w, r)
	})
}
