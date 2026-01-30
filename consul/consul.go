package consul

import (
	"net/url"
	"strconv"
	"time"

	"log/slog"

	capi "github.com/hashicorp/consul/api"
	"github.com/prometheus/common/promslog"
	version "github.com/prometheus/common/version"
)

var (
	serviceName          = "prometheus"
	consulClient         *capi.Client
	checkInterval        = "60s"
	checkTimeout         = "10s"
	checkDeregisterAfter = "30s"
)

type Options struct {
	Url            string
	Token          string
	HealthCheckUrl string
	Enable         bool
}

func initConsul(o *Options) error {
	config := capi.DefaultConfig()
	config.Address = o.Url
	config.Token = o.Token
	client, err := capi.NewClient(config)
	if err != nil {
		return err
	}
	consulClient = client
	return nil

}

func New(logger *slog.Logger, o *Options) {

	if logger == nil {
		logger = promslog.NewNopLogger()
	}

	if err := initConsul(o); err != nil {
		logger.Error("init Consul", "error", err)
		return
	}

	if len(o.HealthCheckUrl) == 0 {
		logger.Error("health check url is empty")
		return
	}

	healthCheckUrl, err := url.Parse(o.HealthCheckUrl)
	if err != nil {
		logger.Error("parse health check url", "error", err)
		return
	}
	host := healthCheckUrl.Host
	port := 9090
	portStr := healthCheckUrl.Port()

	if len(portStr) > 0 {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			logger.Error("parse port from health check url", "error", err)
			return
		}
	} else {
		port = 80
	}

	reg := &capi.AgentServiceRegistration{
		Name:    serviceName,
		Port:    port,
		Address: host[:len(host)-len(":"+strconv.Itoa(port))],
		ID:      host,
		Check: &capi.AgentServiceCheck{
			HTTP:                           o.HealthCheckUrl,
			Interval:                       checkInterval,
			Timeout:                        checkTimeout,
			DeregisterCriticalServiceAfter: checkDeregisterAfter,
		},
		Tags: []string{serviceName},
		Meta: map[string]string{
			"version":    version.Version,
			"start_time": time.Now().Format(time.DateTime),
		},
	}

	err = consulClient.Agent().ServiceRegister(reg)
	if err != nil {
		logger.Error("register service to consul", "error", err)
		return
	}
	logger.Info("register service to consul", "service", serviceName)
}
