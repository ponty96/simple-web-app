package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/kelseyhightower/envconfig"
	"github.com/ponty96/simple-web-app/internal/orders"
	"github.com/ponty96/simple-web-app/internal/rabbitmq"
	"github.com/ponty96/simple-web-app/internal/server"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Environment  string `envconfig:"ENVIRONMENT" default:"local"`
	ListenPort   int    `envconfig:"LISTEN_PORT" default:"4000"`
	ListenHost   string `envconfig:"LISTEN_HOST"`
	Debug        bool   `envconfig:"DEBUG" default:"false"`
	DATABASE_URL string `envconfig:"DATABASE_URL" default:""`
}

func main() {
	var config Config

	err := envconfig.Process("sem", &config)
	if err != nil {
		log.Fatal(err.Error())
	}

	if config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	dbURL := "postgres://postgres:postgres@127.0.0.1:5432/simple-web-app?sslmode=disable"
	if config.DATABASE_URL != "" {
		dbURL = config.DATABASE_URL
	} else {
		log.Warnf("DATABASE_URL not provided; using default %s", dbURL)
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	r := rabbitmq.NewRabbitMQ(rabbitmq.Config{
		URL: "amqp://guest:guest@localhost:5672/",
	})

	defer r.Close()
	p := orders.NewProcessor(conn, r)

	sCfg := server.Config{
		Host:      config.ListenHost,
		Port:      config.ListenPort,
		Processor: p,
	}
	s := server.NewHTTP(&sCfg)

	s.Serve()
}
