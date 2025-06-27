package prometheus

import (
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"

	//pb "github.com/jamesread/japella/gen/protobuf"
	"context"
	//	api "github.com/prometheus/client_golang/api"
	"time"

	promq "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type LocalConfig struct {
	Common      *runtimeconfig.CommonConfig
	PromWatcher *PromWatcherConfig
}

type PromWatcherConfig struct {
	PromUrl string
	Metrics []*PromMetric
}

type PromMetric struct {
	Name string
}

var lastValue int

func getPrometheusValue(a promq.API, q string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, warnings, err := a.Query(ctx, q, time.Now().Add(-time.Hour))

	if err != nil {
		log.Errorf("%+v", err)
		return -1
	}

	if len(warnings) > 0 {
		log.Errorf("Warnings: %v", warnings)
	}

	switch res.Type() {
	case model.ValVector:
		v := res.(model.Vector)

		for _, sample := range v {
			return int(sample.Value)
		}

		log.Warnf("No samples found for metric: %s", q)
		return -1
	default:
		log.Errorf("Unknown type: %s", res.Type())
		return -1
	}
}

func checkPromMetrics(api promq.API, metricName string) {
	log.Infof("Checking metric: %s", metricName)

	val := getPrometheusValue(api, metricName)

	log.Infof("Value: %d", val)

	if val == -1 {
		log.Warnf("Failed to get value for metric: %s", metricName)
		return
	}

	if lastValue == 0 {
		log.Warnf("Last value is 0, skipping")
	}

	if val > lastValue {
		log.Infof("Value increased: %d -> %d", lastValue, val)
	}

	if val == lastValue {
		log.Infof("Value unchanged: %d", val)
	}

	lastValue = val
}

func updateTicker(api promq.API, cfg *PromWatcherConfig) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, metric := range cfg.Metrics {
			checkPromMetrics(api, metric.Name)
		}
	}
}

func main() {
	/*
		runtimeconfig.LoadConfig("config.promwatcher.yaml", cfg.PromWatcher)

		log.Infof("PromWatcherConfig: %+v", cfg.PromWatcher)

		client, err := api.NewClient(api.Config{Address: cfg.PromWatcher.PromUrl})

		if err != nil {
			log.Errorf("%+v", err)
		}

		api := promq.NewAPI(client)

		updateTicker(api, cfg.PromWatcher)
	*/
}
