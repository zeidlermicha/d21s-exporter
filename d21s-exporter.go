package main

import (
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/zeidlermicha/d21s-exporter/collector"
	"github.com/zeidlermicha/go-d21s/pkg/client"
	"github.com/zeidlermicha/go-d21s/pkg/config"
	"io"
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

	// returns nil if not provided and falls back to simple TCP.
	d21sConfig, err := config.NewConfiguration(*d21sUri, *d21sAuthUri, *d21sEmail, *d21sClientKey, *d21sClientPrivateKey)
	if err != nil {
		_ = level.Error(logger).Log(
			"msg", "failed creating d21s client",
			"err", err,
		)
	}
	d21sClient, err := client.NewClient(d21sConfig)
	if err != nil {
		_ = level.Error(logger).Log(
			"msg", "failed creating d21s client",
			"err", err,
		)
	}
	// version metric
	versionMetric := version.NewCollector(Name)
	prometheus.MustRegister(versionMetric)
	prometheus.MustRegister(collector.NewD21SCollector(logger, d21sClient))

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte(`<html>
			<head><title>D21S Exporter</title></head>
			<body>
			<h1>D21S Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			_ = level.Error(logger).Log(
				"msg", "failed handling writer",
				"err", err,
			)
		}
	})

	_ = level.Info(logger).Log(
		"msg", "starting d21s_exporter",
		"addr", *listenAddress,
	)

	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		_ = level.Error(logger).Log(
			"msg", "http server quit",
			"err", err,
		)
	}
}
func getLogger(loglevel, logoutput, logfmt string) log.Logger {
	var out *os.File
	switch strings.ToLower(logoutput) {
	case "stderr":
		out = os.Stderr
	case "stdout":
		out = os.Stdout
	default:
		out = os.Stdout
	}
	var logCreator func(io.Writer) log.Logger
	switch strings.ToLower(logfmt) {
	case "json":
		logCreator = log.NewJSONLogger
	case "logfmt":
		logCreator = log.NewLogfmtLogger
	default:
		logCreator = log.NewLogfmtLogger
	}

	// create a logger
	logger := logCreator(log.NewSyncWriter(out))

	// set loglevel
	var loglevelFilterOpt level.Option
	switch strings.ToLower(loglevel) {
	case "debug":
		loglevelFilterOpt = level.AllowDebug()
	case "info":
		loglevelFilterOpt = level.AllowInfo()
	case "warn":
		loglevelFilterOpt = level.AllowWarn()
	case "error":
		loglevelFilterOpt = level.AllowError()
	default:
		loglevelFilterOpt = level.AllowInfo()
	}
	logger = level.NewFilter(logger, loglevelFilterOpt)
	logger = log.With(logger,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)
	return logger
}
