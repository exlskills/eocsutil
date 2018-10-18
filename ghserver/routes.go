package ghserver

import (
	"github.com/gorilla/mux"
	"net/http"
)

func createRouter() http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)

	// V1 Routes
	v1Router := router.PathPrefix("/v1").Subrouter()
	v1Router.HandleFunc("/", index).Methods("GET")
	v1Router.HandleFunc("/github/repo-push-event", repoPushEventWebhook).Methods("POST")

	return Use(router.ServeHTTP, RecoverAndLog)
}
