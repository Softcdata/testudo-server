package config

import (
	"sort"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
)

// DisasterConfigDTO is the data transfer object for DisasterConfig
type DisasterConfigDTO struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	Labels            map[string]string       `json:"labels,omitempty"`
	CreationTimestamp common.LocalTime        `json:"creation_timestamp"`
	Spec              DisasterConfigSpecDTO   `json:"spec"`
	Status            DisasterConfigStatusDTO `json:"status"`
}

type DisasterConfigNameDTO struct {
	Name          string             `json:"name"`
	ID            string             `json:"id"`
	SourceCluster string             `json:"sourceCluster"`
	TargetCluster string             `json:"targetCluster"`
	Status        dapisv1.StatusType `json:"status"`
}

type DisasterConfigSpecDTO struct {
	SourceCluster       string                 `json:"sourceCluster"`
	TargetCluster       string                 `json:"targetCluster"`
	StorageRepository   string                 `json:"storageRepository"`
	DataSyncType        string                 `json:"dataSyncType"`
	ResourcesSyncPolicy string                 `json:"resourcesSyncPolicy"`
	ResourceSyncPolicy  string                 `json:"resourceSyncPolicy"`
	DataSyncPolicy      string                 `json:"dataSyncPolicy"`
	ImageRewrite        *ImageRewriteConfigDTO `json:"imageRewrite,omitempty"`
	// Enhanced fields for frontend display
	DataSyncCron     string `json:"dataSyncCron"`
	ResourceSyncCron string `json:"resourceSyncCron"`
}

type ImageSourceMappingDTO struct {
	SourceImageSource string `json:"sourceImageSource"`
	TargetImageSource string `json:"targetImageSource"`
}

type ImageRewriteConfigDTO struct {
	Enabled         bool                    `json:"enabled,omitempty"`
	ApplyTo         []string                `json:"applyTo,omitempty"`
	UnmatchedPolicy string                  `json:"unmatchedPolicy,omitempty"`
	Mappings        []ImageSourceMappingDTO `json:"mappings,omitempty"`
}

type DisasterConfigStatusDTO struct {
	Status  dapisv1.StatusType `json:"status,omitempty"`
	Reason  string             `json:"reason,omitempty"`
	Message string             `json:"message,omitempty"`
}

func ConvertToDisasterConfigDTO(item *dapisv1.DisasterConfig) DisasterConfigDTO {
	return DisasterConfigDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Description:       item.Annotations["testudo.softcdata.com/description"],
		Labels:            item.Labels,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
	}
}

func ConvertSpecToDTO(spec dapisv1.DisasterConfigSpec) DisasterConfigSpecDTO {
	return DisasterConfigSpecDTO{
		SourceCluster:       spec.SourceCluster,
		TargetCluster:       spec.TargetCluster,
		StorageRepository:   spec.StorageRepository,
		DataSyncType:        spec.DataSyncType,
		ResourcesSyncPolicy: spec.ResourceSyncPolicy,
		ResourceSyncPolicy:  spec.ResourceSyncPolicy,
		DataSyncPolicy:      spec.DataSyncPolicy,
		ImageRewrite:        convertImageRewriteToDTO(spec.ImageRewrite),
	}
}

func ConvertStatusToDTO(status dapisv1.DisasterConfigStatus) DisasterConfigStatusDTO {
	return DisasterConfigStatusDTO{
		Status:  status.Status,
		Reason:  status.Reason,
		Message: status.Message,
	}
}

// CreateDisasterConfigRequest defines the request body for creating a DisasterConfig
type CreateDisasterConfigRequest struct {
	Name                string                     `json:"name" binding:"required"`
	Description         string                     `json:"description,omitempty"`
	SourceCluster       string                     `json:"sourceCluster" binding:"required"`
	TargetCluster       string                     `json:"targetCluster" binding:"required"`
	StorageRepository   string                     `json:"storageRepository" binding:"required"`
	DataSyncType        string                     `json:"dataSyncType" binding:"required"`
	ResourcesSyncPolicy *string                    `json:"resourcesSyncPolicy,omitempty"`
	ResourceSyncPolicy  *string                    `json:"resourceSyncPolicy,omitempty"`
	DataSyncPolicy      *string                    `json:"dataSyncPolicy,omitempty"`
	ImageRewrite        *ImageRewriteConfigRequest `json:"imageRewrite,omitempty"`
	ImageSources        map[string]string          `json:"imageSources,omitempty"`
}

// ToCRD converts the request DTO to the Operator's DisasterConfigSpec
func (r *CreateDisasterConfigRequest) ToCRD() (dapisv1.DisasterConfigSpec, error) {
	dataSyncPolicy := normalizeOptionalStringPtr(r.DataSyncPolicy)
	resourceSyncPolicy := normalizeOptionalStringPtr(coalesceResourceSyncPolicyPtr(r.ResourcesSyncPolicy, r.ResourceSyncPolicy))
	return dapisv1.DisasterConfigSpec{
		SourceCluster:      r.SourceCluster,
		TargetCluster:      r.TargetCluster,
		StorageRepository:  r.StorageRepository,
		DataSyncType:       r.DataSyncType,
		ResourceSyncPolicy: resourceSyncPolicy,
		DataSyncPolicy:     dataSyncPolicy,
		ImageRewrite:       convertImageRewriteRequestToCRD(r.ImageRewrite),
	}, nil
}

func (r *CreateDisasterConfigRequest) EffectiveImageRewrite() *ImageRewriteConfigRequest {
	return coalesceImageRewriteRequest(r.ImageRewrite, r.ImageSources)
}

// UpdateDisasterConfigRequest defines the request body for updating a DisasterConfig.
// Resource identity comes from URL path param `:name`, not request body.
type UpdateDisasterConfigRequest struct {
	Description         *string                    `json:"description,omitempty"`
	SourceCluster       string                     `json:"sourceCluster,omitempty"`
	TargetCluster       string                     `json:"targetCluster,omitempty"`
	StorageRepository   string                     `json:"storageRepository,omitempty"`
	DataSyncType        string                     `json:"dataSyncType,omitempty"`
	ResourcesSyncPolicy *string                    `json:"resourcesSyncPolicy,omitempty"`
	ResourceSyncPolicy  *string                    `json:"resourceSyncPolicy,omitempty"`
	DataSyncPolicy      *string                    `json:"dataSyncPolicy,omitempty"`
	ImageRewrite        *ImageRewriteConfigRequest `json:"imageRewrite,omitempty"`
	ImageSources        map[string]string          `json:"imageSources,omitempty"`
}

// MergeToCRD updates the existing DisasterConfigSpec with fields from the request
func (r *UpdateDisasterConfigRequest) MergeToCRD(spec *dapisv1.DisasterConfigSpec) error {
	if r.SourceCluster != "" {
		spec.SourceCluster = r.SourceCluster
	}
	if r.TargetCluster != "" {
		spec.TargetCluster = r.TargetCluster
	}
	if r.StorageRepository != "" {
		spec.StorageRepository = r.StorageRepository
	}
	if r.DataSyncType != "" {
		spec.DataSyncType = r.DataSyncType
	}
	if resourceSyncPolicy := coalesceResourceSyncPolicyPtr(r.ResourcesSyncPolicy, r.ResourceSyncPolicy); resourceSyncPolicy != nil {
		spec.ResourceSyncPolicy = normalizeOptionalStringPtr(resourceSyncPolicy)
	}
	if r.DataSyncPolicy != nil {
		spec.DataSyncPolicy = normalizeOptionalStringPtr(r.DataSyncPolicy)
	}
	return nil
}

func (r *UpdateDisasterConfigRequest) EffectiveImageRewrite() *ImageRewriteConfigRequest {
	return coalesceImageRewriteRequest(r.ImageRewrite, r.ImageSources)
}

type ImageSourceMappingRequest struct {
	SourceImageSource string `json:"sourceImageSource"`
	TargetImageSource string `json:"targetImageSource"`
}

type ImageRewriteConfigRequest struct {
	Enabled         bool                        `json:"enabled,omitempty"`
	ApplyTo         []string                    `json:"applyTo,omitempty"`
	UnmatchedPolicy string                      `json:"unmatchedPolicy,omitempty"`
	Mappings        []ImageSourceMappingRequest `json:"mappings,omitempty"`
}

func convertImageRewriteToDTO(in *dapisv1.ImageRewriteConfig) *ImageRewriteConfigDTO {
	if in == nil {
		return nil
	}
	applyTo := make([]string, 0, len(in.ApplyTo))
	for i := range in.ApplyTo {
		applyTo = append(applyTo, string(in.ApplyTo[i]))
	}
	mappings := make([]ImageSourceMappingDTO, 0, len(in.Mappings))
	for i := range in.Mappings {
		mappings = append(mappings, ImageSourceMappingDTO{
			SourceImageSource: in.Mappings[i].SourceImageSource,
			TargetImageSource: in.Mappings[i].TargetImageSource,
		})
	}
	unmatchedPolicy := string(in.UnmatchedPolicy)
	if unmatchedPolicy == "" {
		unmatchedPolicy = string(dapisv1.ImageRewriteUnmatchedPolicyFail)
	}
	return &ImageRewriteConfigDTO{
		Enabled:         in.Enabled,
		ApplyTo:         applyTo,
		UnmatchedPolicy: unmatchedPolicy,
		Mappings:        mappings,
	}
}

func convertImageRewriteRequestToCRD(in *ImageRewriteConfigRequest) *dapisv1.ImageRewriteConfig {
	if in == nil {
		return nil
	}
	applyTo := make([]dapisv1.ImageRewriteApplyTarget, 0, len(in.ApplyTo))
	for i := range in.ApplyTo {
		applyTo = append(applyTo, dapisv1.ImageRewriteApplyTarget(in.ApplyTo[i]))
	}
	mappings := make([]dapisv1.ImageSourceMapping, 0, len(in.Mappings))
	for i := range in.Mappings {
		mappings = append(mappings, dapisv1.ImageSourceMapping{
			SourceImageSource: in.Mappings[i].SourceImageSource,
			TargetImageSource: in.Mappings[i].TargetImageSource,
		})
	}
	return &dapisv1.ImageRewriteConfig{
		Enabled:         in.Enabled,
		ApplyTo:         applyTo,
		UnmatchedPolicy: dapisv1.ImageRewriteUnmatchedPolicy(in.UnmatchedPolicy),
		Mappings:        mappings,
	}
}

func coalesceResourceSyncPolicy(plural, singular string) string {
	plural = strings.TrimSpace(plural)
	if plural != "" {
		return plural
	}
	return strings.TrimSpace(singular)
}

func coalesceResourceSyncPolicyPtr(plural, singular *string) *string {
	if plural != nil {
		return plural
	}
	return singular
}

func normalizeOptionalStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func coalesceImageRewriteRequest(primary *ImageRewriteConfigRequest, imageSources map[string]string) *ImageRewriteConfigRequest {
	if primary != nil {
		return primary
	}
	if len(imageSources) == 0 {
		return nil
	}

	keys := make([]string, 0, len(imageSources))
	for source := range imageSources {
		keys = append(keys, source)
	}
	sort.Strings(keys)

	mappings := make([]ImageSourceMappingRequest, 0, len(keys))
	for _, source := range keys {
		sourceAlias := strings.TrimSpace(source)
		targetAlias := strings.TrimSpace(imageSources[source])
		if sourceAlias == "" || targetAlias == "" {
			continue
		}
		mappings = append(mappings, ImageSourceMappingRequest{
			SourceImageSource: sourceAlias,
			TargetImageSource: targetAlias,
		})
	}
	if len(mappings) == 0 {
		return nil
	}

	return &ImageRewriteConfigRequest{
		Enabled: true,
		ApplyTo: []string{
			string(dapisv1.ImageRewriteApplyResourceSync),
			string(dapisv1.ImageRewriteApplyDrill),
		},
		UnmatchedPolicy: string(dapisv1.ImageRewriteUnmatchedPolicyFail),
		Mappings:        mappings,
	}
}
