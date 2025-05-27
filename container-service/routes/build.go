package routes

import (
	"container-service/builder"
	"container-service/types"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// ----------- Prometheus metrics -----------

// Counter to track total build requests received
var buildRequestCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "build_requests_total",
		Help: "Total number of Docker image build requests received",
	},
)

// Histogram to measure duration of build request handling
var buildRequestDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "build_request_duration_seconds",
		Help:    "Duration of Docker image build request handling in seconds",
		Buckets: prometheus.DefBuckets,
	},
)

func init() {
	// Register the metrics with Prometheus's default registry
	prometheus.MustRegister(buildRequestCounter)
	prometheus.MustRegister(buildRequestDuration)
}

// HandleBuildRequest builds docker image using given data
func HandleBuildRequest(c *gin.Context) {
	// Increment counter for each build request received
	buildRequestCounter.Inc()

	// Start timer for request duration measurement
	timer := prometheus.NewTimer(buildRequestDuration)
	defer timer.ObserveDuration()

	var request types.BuildRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "bad request",
			"error":   err.Error(),
		})
		return
	}

	cloneBaseDir := os.Getenv("CLONE_BASE_DIR")
	log.Println("clone base dir: ", cloneBaseDir)
	if cloneBaseDir == "" {
		cloneBaseDir = "/temp"
	}
	cloneBaseDir, _ = filepath.Abs(cloneBaseDir)
	clonePath := filepath.Join(cloneBaseDir, request.RepoOwner, request.RepoName)
	log.Println("clone path: ", clonePath)

	imageTag := fmt.Sprintf("%s/%s", request.RepoOwner, request.RepoName)
	msg, logs, err := builder.BuildImageFromDockerfile(
		imageTag,
		clonePath,
		request.Dockerfile,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
			"error":   err.Error(),
		})
		return
	}

	if err = os.RemoveAll(clonePath); err != nil {
		log.Println("failed to delete clonePath: ", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   msg,
		"image_tag": imageTag,
		"logs":      logs,
	})
}
