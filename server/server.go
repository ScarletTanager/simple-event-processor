package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/ScarletTanager/algorithms/graph"
)

type Event struct {
	Timestamp float64 `json:"date"`
	Service   string  `json:"service"`
	Target    string  `json:"target"`
}

// HandleEvent - processes a JSON-formatted event upon receipt
func SetupHandleEventHandler(eventChannel chan Event) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Just print it out for now
		body, _ := io.ReadAll(r.Body)
		event := Event{}
		events := []Event{}

		if err := json.Unmarshal(body, &event); err != nil {
			if err := json.Unmarshal(body, &events); err != nil {
				log.Printf("Error %v unmarshaling request body %s", err, string(body))
				w.WriteHeader(http.StatusBadRequest)
				return
			} else {
				event = events[0]
			}
		}

		eventChannel <- event
		log.Printf("Received event for service %s, target %s", event.Service, event.Target)

		w.WriteHeader(http.StatusOK)
	}
}

type ListServiceResponse struct {
	Services []ListServiceResponseEntry `json:"services"`
}

type ListServiceResponseEntry struct {
	Name         string   `json:"name"`
	Dependencies []string `json:"dependsOn"`
}

func SetupListServicesHandler(reg *ServiceRegistry) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		response := ListServiceResponse{
			Services: make([]ListServiceResponseEntry, 0),
		}

		for i := 0; ; i++ {
			svc, err := reg.services.AtIndex(i)
			if err != nil {
				break
			}

			entry := ListServiceResponseEntry{
				Name:         svc.Get("service").(string),
				Dependencies: make([]string, 0),
			}

			reg.services.SearchBreadthFirst(svc.Index())
			for j := 0; ; j++ {
				possibleDependency, err := reg.services.AtIndex(j)
				if err != nil {
					break
				}

				if possibleDependency.Index() != svc.Index() {
					p, _ := reg.services.Path(svc.Index(), possibleDependency.Index())
					if p != nil {
						entry.Dependencies = append(entry.Dependencies, possibleDependency.Get("service").(string))
					}
				}
			}

			response.Services = append(response.Services, entry)
		}
		w.Header().Set("content-type", "application/json")
		bodyBytes, _ := json.Marshal(response)
		w.Write(bodyBytes)
	}
}

type ServiceRegistry struct {
	services graph.Graph
}

func NewRegistry() *ServiceRegistry {
	vs := make([]graph.Vertex, 0)
	svcs, _ := graph.New(vs)
	return &ServiceRegistry{
		services: svcs,
	}
}

func (r *ServiceRegistry) ProcessEvents(c chan Event) {
	var (
		svc, tgt *graph.Vertex
		event    Event
	)

	for event = range c {
		if event.Service != "" {
			if svc = r.Service(event.Service); svc == nil {
				// Service not in registry, add it
				v := graph.Vertex{
					Attributes: graph.Attributes{
						"service": event.Service,
					},
				}

				r.services.Add(v)
				svc = r.Service(event.Service)
			}

			if event.Target != "" {
				if tgt = r.Service(event.Target); tgt == nil {
					// Service not in registry, add it
					v := graph.Vertex{
						Attributes: graph.Attributes{
							"service": event.Target,
						},
					}

					r.services.Add(v)
					tgt = r.Service(event.Target)
				}

				r.AddDependency(svc, tgt)
			}
		}
	}
}

func (r *ServiceRegistry) Service(service string) *graph.Vertex {
	if r.services != nil {
		matches := r.services.WithAttribute("service", service)
		if matches != nil {
			return matches[0]
		}
	}
	return nil
}

func (r *ServiceRegistry) AddDependency(service, target *graph.Vertex) {
	r.services.LinkUnique(service.Index(), target.Index())
}
