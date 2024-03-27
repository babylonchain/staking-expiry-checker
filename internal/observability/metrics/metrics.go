package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-expiry-checker/internal/utils"
)

type Outcome string

const (
	Success Outcome = "success"
	Error   Outcome = "error"
)

func (O Outcome) String() string {
	return string(O)
}

var (
	once                       sync.Once
	metricsRouter              *chi.Mux
	pollDurationHistogram      *prometheus.HistogramVec
	btcClientDurationHistogram *prometheus.HistogramVec
)

// Init initializes the metrics package.
func Init(metricsPort int) {
	once.Do(func() {
		initMetricsRouter(metricsPort)
		registerMetrics()
	})
}

// initMetricsRouter initializes the metrics router.
func initMetricsRouter(metricsPort int) {
	metricsRouter = chi.NewRouter()
	metricsRouter.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	go func() {
		metricsAddr := fmt.Sprintf(":%d", metricsPort)
		err := http.ListenAndServe(metricsAddr, metricsRouter)
		if err != nil {
			log.Fatal().Err(err).Msgf("error starting metrics server on %s", metricsAddr)
		}
	}()
}

// registerMetrics initializes and register the Prometheus metrics.
func registerMetrics() {
	defaultHistogramBucketsSeconds := []float64{0.1, 0.5, 1, 2.5, 5, 10, 30}
	pollDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "poll_duration_seconds",
			Help:    "Histogram of poll durations in seconds.",
			Buckets: defaultHistogramBucketsSeconds,
		},
		[]string{"status"},
	)

	btcClientDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "btcclient_duration_seconds",
			Help:    "Histogram of btcclient durations in seconds.",
			Buckets: defaultHistogramBucketsSeconds,
		},
		[]string{"function", "status"},
	)

	prometheus.MustRegister(
		pollDurationHistogram,
		btcClientDurationHistogram,
	)
}

func RecordBtcClientMetrics[T any](clientRequest func() (T, error)) (T, error) {
	var result T
	functionName := utils.GetFunctionName(1) // Assuming getFunctionName is implemented to use runtime.Caller

	start := time.Now()

	// Perform the client request
	result, err := clientRequest()
	// Determine the outcome status based on whether an error occurred
	status := Success
	if err != nil {
		status = Error
	}

	// Calculate the duration
	duration := time.Since(start).Seconds()

	// Use WithLabelValues to specify the labels and call Observe to record the duration
	btcClientDurationHistogram.WithLabelValues(functionName, status.String()).Observe(duration)

	return result, err
}
