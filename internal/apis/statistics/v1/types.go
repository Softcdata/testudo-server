package v1

import (
	"github.com/softcdata/testudo-server/internal/common"
)

type StatisticsDTO struct {
	Total      int32             `json:"total"`
	InProgress int32             `json:"inProgress"`
	Completed  int32             `json:"completed"`
	Failed     int32             `json:"failed"`
	Canceled   int32             `json:"canceled"`
	Unknown    int32             `json:"unknown"`
	Period     string            `json:"period,omitempty"`
	StartTime  *common.LocalTime `json:"startTime,omitempty"`
	EndTime    *common.LocalTime `json:"endTime,omitempty"`
}

type InstanceStatisticsDTO struct {
	Total        int32 `json:"total"`
	Protected    int32 `json:"protected"`
	Degraded     int32 `json:"degraded"`    // Mapping for failure states or abnormal states
	FailingOver  int32 `json:"failingOver"` // In-progress transition
	FailingBack  int32 `json:"failingBack"` // In-progress transition
	Active       int32 `json:"active"`      // Failover completed
	Paused       int32 `json:"paused"`
	Initializing int32 `json:"initializing"`
	Pending      int32 `json:"pending"`
	Failed       int32 `json:"failed"`
}

type SuccessRateDTO struct {
	SuccessRate float64            `json:"successRate"`
	Details     SuccessRateDetails `json:"details"`
	Period      string             `json:"period,omitempty"`
}

type SuccessRateDetails struct {
	Completed       int32 `json:"completed"`
	Failed          int32 `json:"failed"`
	TotalExecutions int32 `json:"totalExecutions"`
}

type AutoBackupExecutionSummaryDTO struct {
	Period  string                    `json:"period"`
	Range   string                    `json:"range"`
	Total   int32                     `json:"total"`
	Success AutoBackupExecutionBucket `json:"success"`
	Failed  AutoBackupExecutionBucket `json:"failed"`
	Window  AutoBackupExecutionWindow `json:"window"`
}

type AutoBackupExecutionBucket struct {
	Count   int32 `json:"count"`
	Percent int32 `json:"percent"`
}

type AutoBackupExecutionWindow struct {
	Start common.LocalTime `json:"start"`
	End   common.LocalTime `json:"end"`
}

type TaskProgressTrendDTO struct {
	Type      string                  `json:"type"`
	Scope     string                  `json:"scope"`
	Range     string                  `json:"range"`
	Timezone  string                  `json:"timezone"`
	StartTime common.LocalTime        `json:"startTime"`
	EndTime   common.LocalTime        `json:"endTime"`
	Summary   TaskProgressCountDTO    `json:"summary"`
	Buckets   []TaskProgressBucketDTO `json:"buckets"`
	Series    []TaskProgressSeriesDTO `json:"series"`
	Sources   []TaskProgressSourceDTO `json:"sources"`
}

type TaskProgressCountDTO struct {
	Total      int32 `json:"total"`
	InProgress int32 `json:"inProgress"`
	Completed  int32 `json:"completed"`
	Failed     int32 `json:"failed"`
	Canceled   int32 `json:"canceled"`
	Unknown    int32 `json:"unknown"`
}

type TaskProgressBucketDTO struct {
	Date       string `json:"date"`
	Total      int32  `json:"total"`
	InProgress int32  `json:"inProgress"`
	Completed  int32  `json:"completed"`
	Failed     int32  `json:"failed"`
	Canceled   int32  `json:"canceled"`
	Unknown    int32  `json:"unknown"`
}

type TaskProgressSeriesDTO struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type TaskProgressSourceDTO struct {
	Scope     string `json:"scope"`
	Label     string `json:"label"`
	Total     int32  `json:"total"`
	Completed int32  `json:"completed"`
	Failed    int32  `json:"failed"`
}

type StorageUsageDTO struct {
	TotalBackupCount    int64   `json:"totalBackupCount"`
	UsedSpaceBytes      int64   `json:"usedSpaceBytes"`
	AvailableSpaceBytes int64   `json:"availableSpaceBytes"`
	UsageRate           float64 `json:"usageRate"`
	QuotaBytes          int64   `json:"quotaBytes"`
}
