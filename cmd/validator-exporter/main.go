package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/caarlos0/env/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/archway-network/validator-exporter/pkg/collector"
	"github.com/archway-network/validator-exporter/pkg/config"
	"github.com/archway-network/validator-exporter/pkg/grpc"
	log "github.com/archway-network/validator-exporter/pkg/logger"
)

func main() {
	port := flag.Int("p", 8008, "Server port")
	logLevel := log.LevelFlag()

	flag.Parse()

	log.SetLevel(*logLevel)

	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err.Error())
	}

	_, err := grpc.LatestBlockHeight(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	valsCollector := collector.ValidatorsCollector{
		Cfg: cfg,
	}

	prometheus.MustRegister(valsCollector)

	http.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf(":%d", *port)
	log.Info(fmt.Sprintf("Starting server on addr: %s", addr))
	log.Fatal(http.ListenAndServe(addr, nil).Error())
}
