// Package main provides the transceiver_exporter binary.
package main

import (
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"

	"github.com/vexxhost/transceiver_exporter/collector"
	"github.com/vexxhost/transceiver_exporter/pkg/moduleeeprom"
)

var (
	metricsPath = kingpin.Flag(
		"web.telemetry-path",
		"Path under which to expose metrics.",
	).Default("/metrics").String()
	toolkitFlags = webflag.AddFlags(kingpin.CommandLine, ":9459")
	interfaces   = kingpin.Flag(
		"interface",
		"Interface to scrape. May be specified multiple times. Defaults to physical non-loopback interfaces.",
	).Strings()
)

func main() {
	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)

	kingpin.Version(version.Print("transceiver_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promslog.New(promslogConfig)

	logger.Info("Starting transceiver_exporter", "version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())

	reader, err := moduleeeprom.New()
	if err != nil {
		logger.Error("failed to open ethtool client", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			logger.Error("failed to close ethtool client", "err", err)
		}
	}()

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collector.NewTransceiverCollector(reader, *interfaces, logger),
	)

	http.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	if *metricsPath != "/" && *metricsPath != "" {
		landingConfig := web.LandingConfig{
			Name:        "Transceiver Exporter",
			Description: "Prometheus Exporter for transceiver EEPROM telemetry",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			logger.Error("failed to create landing page", "err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	server := &http.Server{}
	if err := web.ListenAndServe(server, toolkitFlags, logger); err != nil {
		logger.Error("error starting HTTP server", "err", err)
		os.Exit(1)
	}
}
