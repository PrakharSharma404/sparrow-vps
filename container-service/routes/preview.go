package routes

import (
	"container-service/preview"
	"container-service/types"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// ----------- Prometheus metrics -----------

// Counter for total preview requests received
var previewRequestCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "preview_requests_total",
		Help: "Total number of Dockerfile preview requests received",
	},
)

// Histogram for measuring duration of preview request processing
var previewRequestDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "preview_request_duration_seconds",
		Help:    "Duration of Dockerfile preview request processing in seconds",
		Buckets: prometheus.DefBuckets,
	},
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(previewRequestCounter)
	prometheus.MustRegister(previewRequestDuration)
}

// HandlePreviewRequest generates a Dockerfile preview based on query params
func HandlePreviewRequest(c *gin.Context) {
	previewRequestCounter.Inc()             // Increment request count
	timer := prometheus.NewTimer(previewRequestDuration) // Start timer
	defer timer.ObserveDuration()            // Observe duration at return

	var params types.PreviewRequest
	if err := c.BindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "bad request",
			"error":   err.Error(),
		})
		return
	}

	fmt.Printf("project type: %s\n", params.ProjectType)
	fmt.Printf("install command: %s\n", params.InstallCommand)
	fmt.Printf("environment vars: %s\n", params.EnvironmentVars)

	var dockerfile string
	var err error

	switch params.ProjectType {
	case "javascript":
		dockerfile, err = preview.GetNodeJSDockerfilePreview(
			params.NodeVersion,
			params.InstallCommand,
			params.BuildCommand,
			params.OutputDirectory,
			params.EnvironmentVars,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "python":
		dockerfile, err = preview.GetPythonDockerfilePreview(
			params.InstallCommand,
			params.ExposePort,
			params.DeployCommand,
			params.EnvironmentVars,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project type"})
		return
	}

	c.String(http.StatusOK, dockerfile)
}

