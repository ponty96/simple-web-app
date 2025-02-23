package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ponty96/simple-web-app/internal/orders"
	"github.com/ponty96/simple-web-app/internal/rabbitmq"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Host      string
	Port      int
	MQ        rabbitmq.MQ
	Processor orders.Processor
}

type Response struct {
	Message string            `json:"message"`
	Code    int               `json:"status_code"`
	Errs    map[string]string `json:"errs"`
	Data    []orders.Order    `json:"data"`
}

type server struct {
	Config *Config
}

func NewHTTP(config *Config) *server {
	return &server{Config: config}
}

func (s *server) Serve() {
	r := mux.NewRouter()
	fmt.Printf("Starting HTTP Server on port %d", s.Config.Port)
	// add the handler for the Order webhook here
	// r.HandleFunc("/show-profile/{profileId}", s.showProfile).Methods("GET")
	// r.HandleFunc("/publish-event", s.publishEventHandler).Methods("POST")

	r.HandleFunc("/webhooks/orders", s.orderWebhookHandler).Methods("POST")
	r.HandleFunc("/orders/{user_id}", s.listUserOrders).Methods("GET")
	r.HandleFunc("/health-check", s.healthCheckHandler).Methods("GET")
	http.ListenAndServe(fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port), r)
}

func (s *server) healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(`:)`))
}

func httpWriteJSON(w http.ResponseWriter, r Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(r.Code)
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		log.Errorf("Failed to write response to client: %s", err)
	}
}
