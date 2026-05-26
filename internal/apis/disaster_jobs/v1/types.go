package jobs

import (
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
)

// DisasterJobDTO is the data transfer object for DisasterJob
type DisasterJobDTO struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Namespace         string               `json:"namespace"`
	Labels            map[string]string    `json:"labels,omitempty"`
	CreationTimestamp common.LocalTime     `json:"creation_timestamp"`
	Spec              DisasterJobSpecDTO   `json:"spec"`
	Status            DisasterJobStatusDTO `json:"status"`
}

type DisasterJobSpecDTO struct {
	DisasterBackup string           `json:"disasterBackup"`
	SyncType       dapisv1.SyncType `json:"syncType,omitempty"`
	ScheduleType   string           `json:"scheduleType,omitempty"`
}

type DisasterJobStatusDTO struct {
	Phase      dapisv1.PhaseType       `json:"phase"`
	Reason     string                  `json:"reason,omitempty"`
	StartTime  common.LocalTime        `json:"startTime,omitempty"`
	Conditions []common.LocalCondition `json:"conditions,omitempty"`
}

func ConvertToDisasterJobDTO(item *dapisv1.DisasterJob) DisasterJobDTO {
	return DisasterJobDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		Labels:            item.Labels,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
	}
}

func ConvertSpecToDTO(spec dapisv1.DisasterJobSpec) DisasterJobSpecDTO {
	return DisasterJobSpecDTO{
		DisasterBackup: spec.DisasterBackup,
		SyncType:       spec.SyncType,
		ScheduleType:   spec.ScheduleType,
	}
}

func ConvertStatusToDTO(status dapisv1.DisasterJobStatus) DisasterJobStatusDTO {
	return DisasterJobStatusDTO{
		Phase:      status.Phase,
		Reason:     status.Reason,
		StartTime:  common.NewLocalTime(status.StartTime),
		Conditions: common.NewLocalConditions(status.Conditions),
	}
}

type CreateDisasterJobRequest struct {
	Name           string           `json:"name"`
	DisasterBackup string           `json:"disasterBackup"`
	SyncType       dapisv1.SyncType `json:"syncType,omitempty"`
	ScheduleType   string           `json:"scheduleType,omitempty"`
}

func (r *CreateDisasterJobRequest) ToCRD() dapisv1.DisasterJobSpec {
	return dapisv1.DisasterJobSpec{
		DisasterBackup: r.DisasterBackup,
		SyncType:       r.SyncType,
		ScheduleType:   r.ScheduleType,
	}
}
