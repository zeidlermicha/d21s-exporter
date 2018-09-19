package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/zeidlermicha/go-d21s/pkg/client"

	"fmt"
	"time"
)

const (
	namespace     = "d21s"
	dataconnector = "dataconnector"
	project       = "project"
)

type D21SCollector struct {
	logger              log.Logger
	d21sClient          *client.Client
	dataConnectorMetric []*DataConnectorMetric
	projectMetric       []*ProjectMetric
}

func NewD21SCollector(logger log.Logger, d21sClient *client.Client) *D21SCollector {
	return &D21SCollector{
		logger:     logger,
		d21sClient: d21sClient,
		dataConnectorMetric: []*DataConnectorMetric{
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, dataconnector, "success_count"), "Number of successfull processed events within the last 24 hours.", []string{"name"}, nil),
				Value: func(dataconnectorMetrics *client.Metrics) float64 {
					return float64(dataconnectorMetrics.SuccessCount)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, dataconnector, "error_count"), "Number of failed processed events within the last 24 hours.", []string{"name"}, nil),
				Value: func(dataconnectorMetrics *client.Metrics) float64 {
					return float64(dataconnectorMetrics.ErrorCount)
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, dataconnector, "latency_99_p"), "The 99th percentile latency of events sent within the last 24 hours.", []string{"name"}, nil),
				Value: func(dataconnectorMetrics *client.Metrics) float64 {
					duration, err := time.ParseDuration(dataconnectorMetrics.Latency99p)
					if err != nil {
						return 0
					}
					return duration.Seconds()
				},
			},
		},
		projectMetric: []*ProjectMetric{
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, project, "sensor_count"), "The number of sensors within the Project.", []string{"name"}, nil),
				Value: func(project *client.Project) int32 {
					return project.SensorCount
				},
			},
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, project, "cloud_connector_count"), "The number of Cloud Connectors within the Project.", []string{"name"}, nil),
				Value: func(project *client.Project) int32 {
					return project.CloudConnectorCount
				},
			},
		},
	}
}

type DataConnectorMetric struct {
	Type  prometheus.ValueType
	Desc  *prometheus.Desc
	Value func(dataconnectorMetrics *client.Metrics) float64
}

type ProjectMetric struct {
	Type  prometheus.ValueType
	Desc  *prometheus.Desc
	Value func(project *client.Project) int32
}

func (c *D21SCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.dataConnectorMetric {
		ch <- metric.Desc
	}
	for _, metric := range c.projectMetric {
		ch <- metric.Desc
	}
}
func (c *D21SCollector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Debug("Started Collecting")
	projects, err := c.d21sClient.ProjectService.GetProjects()
	if err != nil {
		c.logger.WithError(err).Error("error getting projects")
		return
	}
	c.logger.Debug(fmt.Sprintf("found %d projects", len(projects.Projects)))
	for _, project := range projects.Projects {
		for _, metric := range c.projectMetric {
			ch <- prometheus.MustNewConstMetric(metric.Desc, metric.Type, float64(metric.Value(project)), project.DisplayName)
		}
		connectors, err := c.d21sClient.DataConnectorService.GetDataconnectorsByPath(project.Name)
		if err != nil {
			c.logger.WithError(err).Error("error getting connectors")
			continue
		}
		c.logger.Debug(fmt.Sprintf("Number of connectors for %s:%d", project.Name, len(connectors.DataConnectors)))
		for _, connector := range connectors.DataConnectors {
			connMetric, err := c.d21sClient.DataConnectorService.GetDataconnectorMetricByPath(connector.Name)
			if err != nil {
				c.logger.WithError(err).Error("error getting connector metrics")
				continue
			}
			for _, metric := range c.dataConnectorMetric {
				ch <- prometheus.MustNewConstMetric(metric.Desc, metric.Type, metric.Value(connMetric), connector.DisplayName)
			}
		}

	}
}
