package v1

import (
	"fmt"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (h *StatisticsHandler) Register() {
	path := fmt.Sprintf("backuprestorestatistics.%s", dapisv1.GroupVersion.String())
	statistics := h.Rg.Group(path).Use(h.Mw...)
	{
		statistics.GET("/backups", h.GetBackupStatistics)
		statistics.GET("/restores", h.GetRestoreStatistics)
		statistics.GET("/operations", h.GetOperationStatistics)
		statistics.GET("/operations/by-time", h.GetOperationStatisticsByTime)
		statistics.GET("/instances", h.GetInstanceStatistics)
		statistics.GET("/backups/success-rate", h.GetBackupSuccessRate)
		statistics.GET("/autobackups/execution-summary", h.GetAutoBackupExecutionSummary)
		statistics.GET("/tasks/progress", h.GetTaskProgressTrend)
		statistics.GET("/storage", h.GetStorageStatistics)
	}
}
