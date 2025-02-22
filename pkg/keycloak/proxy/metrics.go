package proxy

import "github.com/prometheus/client_golang/prometheus"

var (
	certificateRotationMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "proxy_certificate_rotation_total",
			Help: "The total amount of times the certificate has been rotated",
		},
	)
	oauthTokensMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_oauth_tokens_total",
			Help: "A summary of the tokens issuesd, renewed or failed logins",
		},
		[]string{"action"},
	)
	oauthLatencyMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "proxy_oauth_request_latency",
			Help: "A summary of the request latancy for requests against the openid provider, in seconds",
		},
		[]string{"action"},
	)
	latencyMetric = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "proxy_request_duration",
			Help: "A summary of the http request latency for proxy requests, in seconds",
		},
	)
	statusMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_request_status_total",
			Help: "The HTTP requests partitioned by status code",
		},
		[]string{"code", "method"},
	)
)
