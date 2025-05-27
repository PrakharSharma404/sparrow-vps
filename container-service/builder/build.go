package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http" // <-- Needed for HTTP server to expose metrics endpoint
	"os"
	"path/filepath"
	"time"

	"container-service/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	// Prometheus client libraries
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ----------- Prometheus metrics definitions -----------

// CounterVec to count total Docker image builds attempted, labeled by "status" (success/failure/started)
var buildCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "docker_image_builds_total",
		Help: "Total number of Docker image builds attempted",
	},
	[]string{"status"},
)

// Histogram to track duration of Docker image builds in seconds
var buildDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "docker_image_build_duration_seconds",
		Help:    "Duration of Docker image builds in seconds",
		Buckets: prometheus.DefBuckets, // default bucket sizes
	},
)

// ----------- Register metrics and start HTTP metrics server -----------

func init() {
	// Register the defined metrics with the default Prometheus registry
	prometheus.MustRegister(buildCounter)
	prometheus.MustRegister(buildDuration)

	// Start a separate goroutine running an HTTP server that exposes the /metrics endpoint
	go func() {
		// The promhttp.Handler() exposes the registered metrics in Prometheus format at /metrics
		http.Handle("/metrics", promhttp.Handler())

		log.Println("Starting Prometheus metrics server on :2112/metrics")

		// Listen on port 2112 for metrics requests
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Error starting HTTP server: %v", err)
		}
	}()
}

// ----------- Instrumented BuildImageFromDockerfile function -----------

func BuildImageFromDockerfile(
	imageTag string,
	clonePath string,
	dockerfile string,
) (string, string, error) {
	
	// Start a timer to measure duration of the build and ensure it's recorded when function returns
	timer := prometheus.NewTimer(buildDuration)
	defer timer.ObserveDuration()

	// Increment the counter with label "started" each time a build starts
	buildCounter.WithLabelValues("started").Inc()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		// On error, increment failure counter and return
		buildCounter.WithLabelValues("failure").Inc()
		return "", "", err
	}
	defer cli.Close()

	err = os.WriteFile(filepath.Join(
		clonePath, "Dockerfile"),
		[]byte(dockerfile),
		0644,
	)
	if err != nil {
		buildCounter.WithLabelValues("failure").Inc()
		return "", "", fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	buf := new(bytes.Buffer)
	err = utils.CreateTarArchive(clonePath, buf)
	if err != nil {
		buildCounter.WithLabelValues("failure").Inc()
		return "", "", fmt.Errorf("failed to create tar: %w", err)
	}

	buildResponse, err := cli.ImageBuild(
		ctx,
		buf,
		types.ImageBuildOptions{
			Dockerfile: "Dockerfile",
			Tags:       []string{imageTag},
			Remove:     false,
		},
	)
	if err != nil {
		buildCounter.WithLabelValues("failure").Inc()
		return "", "", fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResponse.Body.Close()

	var logBuf bytes.Buffer
	decoder := json.NewDecoder(buildResponse.Body)
	for {
		var rawMessage map[string]interface{}
		err := decoder.Decode(&rawMessage)
		if err != nil {
			if err == io.EOF {
				break
			}
			buildCounter.WithLabelValues("failure").Inc()
			return "", logBuf.String(), fmt.Errorf("error decoding event: %w", err)
		}

		if stream, ok := rawMessage["stream"].(string); ok {
			if stream == "" {
				continue // skip if empty
			}
			timestamp := time.Now().Format("2006-01-02 15:04:05.000")
			logMessage := fmt.Sprintf("[%s] %s", timestamp, stream)
			log.Print(logMessage)
			logBuf.WriteString(logMessage)
		} else {
			var event events.Message
			rawMessageBytes, err := json.Marshal(rawMessage)
			if err != nil {
				log.Printf("Error marshalling message: %v, using default log : %v", err, rawMessage)
				logBuf.WriteString(fmt.Sprintf("Error marshalling message: %v, using default log : %v", err, rawMessage))
				continue
			}
			err = json.Unmarshal(rawMessageBytes, &event)
			if err != nil {
				log.Printf("Error unmarshalling event: %v, using default log : %v", err, rawMessage)
				logBuf.WriteString(fmt.Sprintf("Error unmarshalling event: %v, using default log : %v", err, rawMessage))
				continue
			}
			utils.PrettyPrintEvent(os.Stdout, event)
			utils.PrettyPrintEvent(&logBuf, event)
		}
	}

	// On successful build, increment success counter
	buildCounter.WithLabelValues("success").Inc()

	return "image build complete", logBuf.String(), nil
}

