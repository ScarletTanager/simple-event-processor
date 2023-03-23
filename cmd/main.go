package main

import (
	"log"
	"net/http"

	"github.com/ScarletTanager/simple-event-processor/server"
)

func main() {
	eventChannel := make(chan server.Event, 10)
	reg := server.NewRegistry()
	go reg.ProcessEvents(eventChannel)
	http.HandleFunc("/event", server.SetupHandleEventHandler(eventChannel))
	http.HandleFunc("/services", server.SetupListServicesHandler(reg))
	log.Fatal(http.ListenAndServe(":9000", nil))
}
