package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/ponty96/simple-web-app/internal/server"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Environment string `envconfig:"ENVIRONMENT" default:"local"`
	ListenPort  int    `envconfig:"LISTEN_PORT" default:"4000"`
	ListenHost  string `envconfig:"LISTEN_HOST"`
	Debug       bool   `envconfig:"DEBUG" default:"false"`
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

	sCfg := server.Config{
		Host: config.ListenHost,
		Port: config.ListenPort,
	}
	s := server.NewHTTP(&sCfg)

	s.Serve()
}
