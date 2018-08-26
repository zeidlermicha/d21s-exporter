package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
	"github.com/zeidlermicha/d21s-exporter/collector"
	"github.com/zeidlermicha/go-d21s/pkg/client"
	"github.com/zeidlermicha/go-d21s/pkg/config"
	"net/http"
	"os"
	"strings"
)

func main() {
	var (
		Name                 = "d21s_exporter"
		listenAddress        = flag.String("web.listen-address", ":9108", "Address to listen on for web interface and telemetry.")
		metricsPath          = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		d21sUri              = flag.String("d21s.uri", "https://api.disruptive-technologies.com/v2", "HTTP API address of d21s.")
		d21sAuthUri          = flag.String("d21s.auth", "https://identity.disruptive-technologies.com/oauth2/token", "HTTP API Auth address of d21s.")
		d21sClientPrivateKey = flag.String("d21s.client-private-key", "", "Service Account private key")
		d21sClientKey        = flag.String("d21s.client-key", "", "Service Account public key")
		d21sEmail            = flag.String("d21s.client-mail", "", "Service Account email")
		logLevel             = flag.String("log.level", "info", "Sets the loglevel. Valid levels are debug, info, warn, error")
		logFormat            = flag.String("log.format", "logfmt", "Sets the log format. Valid formats are json and logfmt")
		logOutput            = flag.String("log.output", "stdout", "Sets the log output. Valid outputs are stdout and stderr")
		showVersion          = flag.Bool("version", false, "Show version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Print(version.Print(Name))
		os.Exit(0)
	}

	logger := getLogger(*logLevel, *logOutput, *logFormat)
	d21sConfig, err := config.NewConfiguration(*d21sUri, *d21sAuthUri, *d21sEmail, *d21sClientKey, *d21sClientPrivateKey)
	if err != nil {
		logger.WithError(err).Error("failed creating d21s client")
	}
	d21sClient, err := client.NewClient(d21sConfig)
	if err != nil {
		logger.WithError(err).Error("failed creating d21s client")
	}
	// version metric
	versionMetric := version.NewCollector(Name)
	prometheus.MustRegister(versionMetric)
	prometheus.MustRegister(collector.NewD21SCollector(logger, d21sClient))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte(`<html>
			<head><title>D21S Exporter</title></head>
			<body>
			<h1>D21S Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			logger.WithError(err).Error("failed handling writer")
		}
	})

	logger.Infof("starting d21s_exporter addr: %s", *listenAddress)

	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		logger.WithError(err).Error("http server quit")
	}
}
func getLogger(loglevel, logoutput, logfmt string) log.Logger {
	logger := log.Logger{}
	var out *os.File
	switch strings.ToLower(logoutput) {
	case "stderr":
		out = os.Stderr
	case "stdout":
		out = os.Stdout
	default:
		out = os.Stdout
	}
	switch strings.ToLower(logfmt) {
	case "json":
		logger.Formatter = &log.JSONFormatter{}
	case "logfmt":
		logger.Formatter = &log.TextFormatter{}
	default:
		logger.Formatter = &log.TextFormatter{}

	}

	// create a logger
	logger.Out = out

	// set loglevel
	switch strings.ToLower(loglevel) {
	case "debug":
		logger.Level = log.DebugLevel
	case "info":
		logger.Level = log.InfoLevel
	case "warn":
		logger.Level = log.WarnLevel
	case "error":
		logger.Level = log.ErrorLevel
	case "fatal":
		logger.Level = log.FatalLevel
	case "panic":
		logger.Level = log.PanicLevel
	default:
		logger.Level = log.InfoLevel
	}
	return logger
}
