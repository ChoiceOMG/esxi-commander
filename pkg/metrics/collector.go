package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	VMOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "ceso_vm_operation_duration_seconds",
			Help: "Duration of VM operations",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 90, 120},
		},
		[]string{"operation", "status"},
	)

	VMOperationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ceso_vm_operation_total",
			Help: "Total number of VM operations",
		},
		[]string{"operation", "status"},
	)

	BackupSizeBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ceso_backup_size_bytes",
			Help: "Size of backups in bytes",
		},
		[]string{"vm_name"},
	)

	BackupDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "ceso_backup_duration_seconds",
			Help: "Duration of backup operations",
			Buckets: []float64{10, 30, 60, 120, 300, 600, 900, 1200},
		},
		[]string{"operation"},
	)

	BackupTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ceso_backup_total",
			Help: "Total number of backup operations",
		},
		[]string{"operation", "status"},
	)

	AIAgentOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ceso_ai_agent_operations_total",
			Help: "Total AI agent operations by mode",
		},
		[]string{"mode", "operation"},
	)

	AIAgentPromotionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ceso_ai_agent_promotion_total",
			Help: "Total AI agent promotions",
		},
		[]string{"from", "to"},
	)

	ESXiConnectionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ceso_esxi_connection_total",
			Help: "Total ESXi connection attempts",
		},
		[]string{"status"},
	)

	ESXiConnectionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "ceso_esxi_connection_duration_seconds",
			Help: "Duration of ESXi connection establishment",
			Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
	)
)

func RecordVMOperation(operation string, status string, duration float64) {
	VMOperationDuration.WithLabelValues(operation, status).Observe(duration)
	VMOperationTotal.WithLabelValues(operation, status).Inc()
}

func RecordBackupOperation(operation string, status string, duration float64) {
	BackupDurationSeconds.WithLabelValues(operation).Observe(duration)
	BackupTotal.WithLabelValues(operation, status).Inc()
}

func RecordBackupSize(vmName string, sizeBytes float64) {
	BackupSizeBytes.WithLabelValues(vmName).Set(sizeBytes)
}

func RecordAIOperation(mode string, operation string) {
	AIAgentOperationsTotal.WithLabelValues(mode, operation).Inc()
}

func RecordAIPromotion(from string, to string) {
	AIAgentPromotionTotal.WithLabelValues(from, to).Inc()
}

func RecordESXiConnection(status string, duration float64) {
	ESXiConnectionTotal.WithLabelValues(status).Inc()
	if status == "success" {
		ESXiConnectionDuration.Observe(duration)
	}
}