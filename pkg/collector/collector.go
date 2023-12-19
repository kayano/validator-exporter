package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/archway-network/validator-exporter/pkg/config"
	"github.com/archway-network/validator-exporter/pkg/grpc"
	log "github.com/archway-network/validator-exporter/pkg/logger"
	"github.com/archway-network/validator-exporter/pkg/types"
)

const (
	missedBlocksMetricName = "cosmos_validator_missed_blocks"
)

var missedBlocks = prometheus.NewDesc(
	missedBlocksMetricName,
	"Returns missed blocks for a validator.",
	[]string{
		"valcons",
		"valoper",
		"moniker",
	},
	nil,
)

type ValidatorsCollector struct {
	Cfg config.Config
}

func (vc ValidatorsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- missedBlocks
}

func (vc ValidatorsCollector) Collect(ch chan<- prometheus.Metric) {
	vals, err := grpc.SigningValidators(vc.Cfg)
	if err != nil {
		log.Error(fmt.Sprintf("error getting signing validators: %s", err))
	} else {
		log.Debug("Start collecting", zap.String("metric", missedBlocksMetricName))

		for _, m := range vc.missedBlocksMetrics(vals) {
			ch <- m
		}

		log.Debug("Stop collecting", zap.String("metric", missedBlocksMetricName))
	}
}

func (vc ValidatorsCollector) missedBlocksMetrics(vals []types.Validator) []prometheus.Metric {
	metrics := []prometheus.Metric{}

	for _, v := range vals {
		metrics = append(
			metrics,
			prometheus.MustNewConstMetric(
				missedBlocks,
				prometheus.GaugeValue,
				float64(v.MissedBlocks),
				[]string{
					v.ConsAddress,
					v.OperatorAddress,
					v.Moniker,
				}...,
			),
		)
	}

	return metrics
}
