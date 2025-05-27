package preview

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ----------- Prometheus metrics -----------

// Counter to track how many times GetNodeJSDockerfilePreview is called
var previewCallCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "nodejs_dockerfile_preview_calls_total",
		Help: "Total number of NodeJS Dockerfile preview generations",
	},
)

// Histogram to measure duration of Dockerfile preview generation
var previewDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "nodejs_dockerfile_preview_duration_seconds",
		Help:    "Duration of NodeJS Dockerfile preview generation in seconds",
		Buckets: prometheus.DefBuckets,
	},
)

func init() {
	// Register metrics with Prometheus default registry
	prometheus.MustRegister(previewCallCounter)
	prometheus.MustRegister(previewDuration)
}

// GetNodeJSDockerfilePreview generates a Dockerfile string for NodeJS projects
func GetNodeJSDockerfilePreview(
	nodeVersion string,
	installCommand string,
	buildCommand string,
	outputDirectory string,
	environmentVars string,
) (string, error) {
	// Start timer to track how long this function takes
	timer := prometheus.NewTimer(previewDuration)
	defer timer.ObserveDuration()

	// Increment counter each time this function is called
	previewCallCounter.Inc()

	// Compose the Dockerfile string using fmt.Sprintf
	dockerfile := fmt.Sprintf(
		`FROM node:%v-alpine AS builder%v
WORKDIR /app
COPY package*.json ./
RUN %v
COPY . ./
RUN chmod -R a+x node_modules
RUN %v

FROM nginx:alpine
COPY --from=builder /app/%v /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]`,
		nodeVersion,
		func() string {
			if environmentVars != "" {
				return "\nENV " + environmentVars
			}
			return ""
		}(),
		installCommand,
		buildCommand,
		outputDirectory,
	)

	return dockerfile, nil
}

