package preview

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ----------- Prometheus metrics -----------

// Counter to track how many times GetPythonDockerfilePreview is called
var previewCallCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "python_dockerfile_preview_calls_total",
		Help: "Total number of Python Dockerfile preview generations",
	},
)

// Histogram to measure duration of Dockerfile preview generation
var previewDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "python_dockerfile_preview_duration_seconds",
		Help:    "Duration of Python Dockerfile preview generation in seconds",
		Buckets: prometheus.DefBuckets,
	},
)

func init() {
	// Register metrics with Prometheus default registry
	prometheus.MustRegister(previewCallCounter)
	prometheus.MustRegister(previewDuration)
}

// GetPythonDockerfilePreview generates a Dockerfile string for Python projects
func GetPythonDockerfilePreview(
	installCommand string,
	exposePort string,
	deployCommand string,
	environmentVars string,
) (string, error) {
	// Start timer to track how long this function takes
	timer := prometheus.NewTimer(previewDuration)
	defer timer.ObserveDuration()

	// Increment counter each time this function is called
	previewCallCounter.Inc()

	// Convert deployCommand string into a JSON-like array string for CMD
	cmdParts := strings.Split(deployCommand, " ")
	cmdString := `[`
	for i, part := range cmdParts {
		cmdString += fmt.Sprintf(`"%s"`, part)
		if i < len(cmdParts)-1 {
			cmdString += `, `
		}
	}
	cmdString += `]`

	// Compose the Dockerfile string using fmt.Sprintf
	dockerfile := fmt.Sprintf(
		`FROM python:alpine%v
WORKDIR /app
COPY requirements.txt ./
RUN %v
COPY . ./
EXPOSE %v
CMD %v`,
		func() string {
			if environmentVars != "" {
				return "\nENV " + environmentVars
			}
			return ""
		}(),
		installCommand,
		exposePort,
		cmdString,
	)

	return dockerfile, nil
}
