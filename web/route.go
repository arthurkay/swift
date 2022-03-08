package web

import (
	"petricoh/web/handlers"

	"github.com/gorilla/mux"
)

func Routes() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	r.HandleFunc("/status", handlers.Status).Methods("POST")
	r.HandleFunc("/reboot", handlers.Reboot).Methods("POST")
	r.HandleFunc("/shutdown", handlers.ShutDown).Methods("POST")
	r.HandleFunc("/startup", handlers.StartUp).Methods("POST")
	r.HandleFunc("/install", handlers.Install).Methods("POST")
	return r
}
