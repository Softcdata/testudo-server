package storage

import (
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	corev1 "k8s.io/api/core/v1"
)

// DisasterStorageDTO is the data transfer object for StorageRepository
type DisasterStorageDTO struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	Namespace         string                   `json:"namespace"`
	Labels            map[string]string        `json:"labels,omitempty"`
	CreationTimestamp common.LocalTime         `json:"creation_timestamp"`
	Spec              DisasterStorageSpecDTO   `json:"spec"`
	Status            DisasterStorageStatusDTO `json:"status"`
}

type DisasterStorageSpecDTO struct {
	StorageType     string                                   `json:"storageType"`
	Bucket          string                                   `json:"bucket"`
	Region          string                                   `json:"region"`
	Endpoint        string                                   `json:"endpoint"`
	AddressingStyle dapisv1.StorageRepositoryAddressingStyle `json:"addressingStyle"`
	CAConfigured    bool                                     `json:"caConfigured"`
	CASecretRef     *corev1.LocalObjectReference             `json:"caSecretRef,omitempty"`
	QuotaBytes      int64                                    `json:"quotaBytes,omitempty"`
	// Sensitive fields like AccessKey and SecretKey are excluded
}

type DisasterStorageNameDTO struct {
	Name   string             `json:"name"`
	ID     string             `json:"id"`
	Status dapisv1.StatusType `json:"status"`
}

type DisasterStorageStatusDTO struct {
	Status           dapisv1.StatusType `json:"status"`
	LastCheckTime    *common.LocalTime  `json:"lastCheckTime,omitempty"`
	Reason           string             `json:"reason,omitempty"`
	Message          string             `json:"message,omitempty"`
	UsedSpaceBytes   int64              `json:"usedSpaceBytes,omitempty"`
	TotalBackupCount int64              `json:"totalBackupCount,omitempty"`
}

func ConvertToDisasterStorageDTO(item *dapisv1.StorageRepository) DisasterStorageDTO {
	return DisasterStorageDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		Labels:            item.Labels,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
	}
}

func ConvertSpecToDTO(spec dapisv1.StorageRepositorySpec) DisasterStorageSpecDTO {
	return DisasterStorageSpecDTO{
		StorageType:     spec.StorageType,
		Bucket:          spec.Bucket,
		Region:          spec.Region,
		Endpoint:        spec.Endpoint,
		AddressingStyle: spec.GetAddressingStyle(),
		CAConfigured:    spec.CASecretRef != nil && spec.CASecretRef.Name != "",
		CASecretRef:     spec.CASecretRef,
		QuotaBytes:      spec.QuotaBytes,
	}
}

func ConvertStatusToDTO(status dapisv1.StorageRepositoryStatus) DisasterStorageStatusDTO {
	return DisasterStorageStatusDTO{
		Status:           status.Status,
		LastCheckTime:    common.NewLocalTimePtr(status.LastCheckTime),
		Reason:           status.Reason,
		Message:          status.Message,
		UsedSpaceBytes:   status.UsedSpaceBytes,
		TotalBackupCount: status.TotalBackupCount,
	}
}

// CreateDisasterStorageRequest defines the request body for creating a StorageRepository
type CreateDisasterStorageRequest struct {
	Name            string                                   `json:"name" binding:"required,max=50,min=1"`
	StorageType     string                                   `json:"storageType" binding:"required"`
	Bucket          string                                   `json:"bucket" binding:"required,max=50,min=1"`
	Region          string                                   `json:"region" binding:"required,max=50,min=1"`
	Endpoint        string                                   `json:"endpoint" binding:"required"`
	AccessKey       string                                   `json:"accessKey" binding:"required"`
	SecretKey       string                                   `json:"secretKey" binding:"required"`
	AddressingStyle dapisv1.StorageRepositoryAddressingStyle `json:"addressingStyle,omitempty"`
	CABundle        string                                   `json:"caBundle,omitempty"`
	CASecretRef     *corev1.LocalObjectReference             `json:"caSecretRef,omitempty"`
	QuotaBytes      int64                                    `json:"quotaBytes,omitempty"`
}

// ToCRD converts the request DTO to the Operator's StorageRepositorySpec
func (r *CreateDisasterStorageRequest) ToCRD() dapisv1.StorageRepositorySpec {
	return dapisv1.StorageRepositorySpec{
		StorageType:     r.StorageType,
		Bucket:          r.Bucket,
		Region:          r.Region,
		Endpoint:        r.Endpoint,
		AccessKey:       r.AccessKey,
		SecretKey:       r.SecretKey,
		AddressingStyle: normalizeAddressingStyle(r.AddressingStyle),
		CASecretRef:     r.CASecretRef,
		QuotaBytes:      r.QuotaBytes,
	}
}

// UpdateDisasterStorageRequest defines the request body for updating a StorageRepository
type UpdateDisasterStorageRequest struct {
	Name            string                                    `json:"name" binding:"required"`
	StorageType     string                                    `json:"storageType,omitempty"`
	Bucket          string                                    `json:"bucket,omitempty"`
	Region          string                                    `json:"region,omitempty"`
	Endpoint        string                                    `json:"endpoint,omitempty"`
	AccessKey       string                                    `json:"accessKey,omitempty"`
	SecretKey       string                                    `json:"secretKey,omitempty"`
	AddressingStyle *dapisv1.StorageRepositoryAddressingStyle `json:"addressingStyle,omitempty"`
	CABundle        *string                                   `json:"caBundle,omitempty"`
	CASecretRef     *corev1.LocalObjectReference              `json:"caSecretRef,omitempty"`
	ClearCA         *bool                                     `json:"clearCa,omitempty"`
	QuotaBytes      *int64                                    `json:"quotaBytes,omitempty"` // pointer for tracking partial updates vs zero value
}

// MergeToCRD updates the existing StorageRepositorySpec with fields from the request
func (r *UpdateDisasterStorageRequest) MergeToCRD(spec *dapisv1.StorageRepositorySpec) {
	if r.StorageType != "" {
		spec.StorageType = r.StorageType
	}
	if r.Bucket != "" {
		spec.Bucket = r.Bucket
	}
	if r.Region != "" {
		spec.Region = r.Region
	}
	if r.Endpoint != "" {
		spec.Endpoint = r.Endpoint
	}
	if r.AccessKey != "" {
		spec.AccessKey = r.AccessKey
	}
	if r.SecretKey != "" {
		spec.SecretKey = r.SecretKey
	}
	if r.AddressingStyle != nil {
		spec.AddressingStyle = normalizeAddressingStyle(*r.AddressingStyle)
	}
	if r.CASecretRef != nil {
		spec.CASecretRef = r.CASecretRef
	}
	if r.ClearCA != nil && *r.ClearCA {
		spec.CASecretRef = nil
	}
	if r.QuotaBytes != nil {
		spec.QuotaBytes = *r.QuotaBytes
	}
}

type ValidateS3ConnectionRequest struct {
	Endpoint        string                                   `json:"endpoint" binding:"required"`
	Region          string                                   `json:"region,omitempty"`
	Bucket          string                                   `json:"bucket,omitempty"`
	AccessKey       string                                   `json:"accessKey" binding:"required"`
	SecretKey       string                                   `json:"secretKey" binding:"required"`
	StorageType     string                                   `json:"storageType,omitempty"` // Optional, default to s3
	AddressingStyle dapisv1.StorageRepositoryAddressingStyle `json:"addressingStyle,omitempty"`
	CABundle        string                                   `json:"caBundle,omitempty"`
	CASecretRef     *corev1.LocalObjectReference             `json:"caSecretRef,omitempty"`
}
type PatchStorageRepositoryRequest struct {
	AccessKey       *string                                   `json:"accessKey,omitempty"`
	SecretKey       *string                                   `json:"secretKey,omitempty"`
	Bucket          *string                                   `json:"bucket,omitempty"`
	Region          *string                                   `json:"region,omitempty"`
	AddressingStyle *dapisv1.StorageRepositoryAddressingStyle `json:"addressingStyle,omitempty"`
	CABundle        *string                                   `json:"caBundle,omitempty"`
	CASecretRef     *corev1.LocalObjectReference              `json:"caSecretRef,omitempty"`
	ClearCA         *bool                                     `json:"clearCa,omitempty"`
}

// ValidateConnectivityRequest defines the request body for connectivity check
// ValidateConnectivityRequest defines the request body for connectivity check
type ValidateConnectivityRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	StorageName string `json:"storage_name" binding:"required"`
}

func normalizeAddressingStyle(style dapisv1.StorageRepositoryAddressingStyle) dapisv1.StorageRepositoryAddressingStyle {
	if style == "" {
		return dapisv1.StorageRepositoryAddressingStylePathStyle
	}
	return style
}
