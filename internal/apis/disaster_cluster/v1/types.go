package cluster

import (
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
)

const ClusterTagLabel = "testudo.softcdata.com/cluster-tag"
const ClusterDescriptionAnnotation = "testudo.softcdata.com/description"
const StorageClassDefaultAnnotation = "storageclass.kubernetes.io/is-default-class"
const StorageClassBetaDefaultAnnotation = "storageclass.beta.kubernetes.io/is-default-class"
const IngressClassDefaultAnnotation = "ingressclass.kubernetes.io/is-default-class"

// DisasterClusterDTO is the data transfer object for Cluster
type DisasterClusterDTO struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	Description       string                   `json:"description"`
	Labels            map[string]string        `json:"labels,omitempty"`
	CreationTimestamp common.LocalTime         `json:"creation_timestamp"`
	Spec              DisasterClusterSpecDTO   `json:"spec"`
	Status            DisasterClusterStatusDTO `json:"status"`
}

type DisasterClusterSpecDTO struct {
	HasToken      bool              `json:"has_token"`
	HasKubeConfig bool              `json:"has_kube_config"`
	ImageSources  []ImageSourceDTO  `json:"imageSources,omitempty"`
	VeleroInstall *VeleroInstallDTO `json:"veleroInstall,omitempty"`
}

type ImageSourceDTO struct {
	Name     string `json:"name"`
	Registry string `json:"registry"`
}

type VeleroInstallDTO struct {
	ImageRegistry        string `json:"imageRegistry,omitempty"`
	Username             string `json:"username,omitempty"`
	CredentialConfigured bool   `json:"credentialConfigured"`
}

type VeleroInstallWriteDTO struct {
	ImageRegistry    string `json:"imageRegistry,omitempty"`
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	RemoveCredential bool   `json:"removeCredential,omitempty"`
}

type PatchVeleroInstallWriteDTO struct {
	ImageRegistry    *string `json:"imageRegistry,omitempty"`
	Username         *string `json:"username,omitempty"`
	Password         *string `json:"password,omitempty"`
	RemoveCredential *bool   `json:"removeCredential,omitempty"`
}

type DisasterClusterNameDTO struct {
	Name                   string `json:"name"`
	ID                     string `json:"id"`
	NamespaceCount         int    `json:"namespaceCount,omitempty"`
	ResourceTotalCount     int    `json:"resourceTotalCount,omitempty"`
	WorkloadNamespaceCount int    `json:"workloadNamespaceCount,omitempty"`
	WorkloadTotalCount     int    `json:"workloadTotalCount,omitempty"`
	Tag                    string `json:"tag,omitempty"`
}

type DisasterClusterStatusDTO struct {
	Status                 dapisv1.StatusType `json:"status"`
	Endpoint               string             `json:"endpoint"`
	K8SVersion             string             `json:"k8sVersion"`
	VeleroVersion          string             `json:"veleroVersion"`
	NamespaceCount         int                `json:"namespaceCount,omitempty"`
	ResourceTotalCount     int                `json:"resourceTotalCount,omitempty"`
	NamespaceStats         map[string]int     `json:"namespaceStats,omitempty"`
	WorkloadNamespaceCount int                `json:"workloadNamespaceCount,omitempty"`
	WorkloadTotalCount     int                `json:"workloadTotalCount,omitempty"`
	WorkloadNamespaceStats map[string]int     `json:"workloadNamespaceStats,omitempty"`
	LastCheckTime          *common.LocalTime  `json:"lastCheckTime,omitempty"`
	NodeCount              int                `json:"nodeCount,omitempty"`
	Reason                 string             `json:"reason,omitempty"`
	Message                string             `json:"message,omitempty"`
	TokenExpiration        *common.LocalTime  `json:"tokenExpiration,omitempty"`
}

type RestoreClassItemDTO struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

type RestoreClassListDTO struct {
	TargetCluster  string                `json:"targetCluster"`
	StorageClasses []RestoreClassItemDTO `json:"storageClasses"`
	IngressClasses []RestoreClassItemDTO `json:"ingressClasses"`
}

type ClusterEndpointConflictMeta struct {
	ConflictType     string `json:"conflictType"`
	ConflictCluster  string `json:"conflictCluster"`
	ConflictEndpoint string `json:"conflictEndpoint"`
}

type ClusterProtectedNamespaceOwnerDTO struct {
	InstanceName      string `json:"instanceName"`
	InstanceNamespace string `json:"instanceNamespace"`
	ConfigName        string `json:"configName"`
}

type ClusterProtectedNamespaceDTO struct {
	Namespace string                              `json:"namespace"`
	Cluster   string                              `json:"cluster"`
	Owners    []ClusterProtectedNamespaceOwnerDTO `json:"owners,omitempty"`
}

func ConvertToDisasterClusterDTO(item *dapisv1.Cluster) DisasterClusterDTO {
	return DisasterClusterDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Description:       item.Annotations[ClusterDescriptionAnnotation],
		Labels:            item.Labels,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
	}
}

func ConvertSpecToDTO(spec dapisv1.ClusterSpec) DisasterClusterSpecDTO {
	return DisasterClusterSpecDTO{
		HasToken:      spec.Token != "",
		HasKubeConfig: len(spec.KubeConfig) > 0,
		ImageSources:  convertImageSourcesToDTO(spec.ImageSources),
		VeleroInstall: convertVeleroInstallToDTO(spec.VeleroInstall),
	}
}

func ConvertStatusToDTO(status dapisv1.ClusterStatus) DisasterClusterStatusDTO {
	return DisasterClusterStatusDTO{
		Status:                 status.Status,
		Endpoint:               status.Endpoint,
		K8SVersion:             status.K8SVersion,
		VeleroVersion:          status.VeleroVersion,
		NamespaceCount:         status.NamespaceCount,
		ResourceTotalCount:     status.ResourceTotalCount,
		NamespaceStats:         status.NamespaceStats,
		WorkloadNamespaceCount: status.WorkloadNamespaceCount,
		WorkloadTotalCount:     status.WorkloadTotalCount,
		WorkloadNamespaceStats: status.WorkloadNamespaceStats,
		LastCheckTime:          common.NewLocalTimePtr(status.LastCheckTime),
		NodeCount:              status.NodeCount,
		Reason:                 status.Reason,
		Message:                status.Message,
		TokenExpiration:        common.NewLocalTimePtr(status.TokenExpiration),
	}
}

func isStorageClassDefault(annotations map[string]string) bool {
	if len(annotations) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(annotations[StorageClassDefaultAnnotation]), "true") ||
		strings.EqualFold(strings.TrimSpace(annotations[StorageClassBetaDefaultAnnotation]), "true")
}

func isIngressClassDefault(annotations map[string]string) bool {
	if len(annotations) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(annotations[IngressClassDefaultAnnotation]), "true")
}

// CreateDisasterClusterRequest defines the request body for creating a Cluster
type CreateDisasterClusterRequest struct {
	Name          string                 `json:"name" binding:"required,max=50,min=1"`
	Description   string                 `json:"description,omitempty"`
	Token         string                 `json:"token,omitempty"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	KubeConfig    []byte                 `json:"kubeConfig,omitempty"`
	Tag           string                 `json:"tag,omitempty" binding:"max=200,min=1"`
	ImageSources  []ImageSourceDTO       `json:"imageSources,omitempty"`
	VeleroInstall *VeleroInstallWriteDTO `json:"veleroInstall,omitempty"`
}
type ValidateKubeConfigRequest struct {
	KubeConfig []byte `json:"kubeConfig"`
	Token      string `json:"token"`
	Endpoint   string `json:"endpoint"`
}

// ToCRD converts the request DTO to the Operator's ClusterSpec
func (r *CreateDisasterClusterRequest) ToCRD() dapisv1.ClusterSpec {
	return dapisv1.ClusterSpec{
		Token:         r.Token,
		Endpoint:      r.Endpoint,
		KubeConfig:    r.KubeConfig,
		ImageSources:  convertImageSourcesToCRD(r.ImageSources),
		VeleroInstall: convertVeleroInstallWriteToCRD(r.VeleroInstall),
	}
}

// UpdateDisasterClusterRequest defines the request body for updating a Cluster
type UpdateDisasterClusterRequest struct {
	Name          string                 `json:"name" binding:"required，max=50,min=1"`
	Description   string                 `json:"description,omitempty"`
	Token         string                 `json:"token,omitempty"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	KubeConfig    []byte                 `json:"kubeConfig,omitempty"`
	Tag           string                 `json:"tag,omitempty"`
	ImageSources  []ImageSourceDTO       `json:"imageSources,omitempty"`
	VeleroInstall *VeleroInstallWriteDTO `json:"veleroInstall,omitempty"`
}

// MergeToCRD updates the existing ClusterSpec with fields from the request
func (r *UpdateDisasterClusterRequest) MergeToCRD(spec *dapisv1.ClusterSpec) {
	if r.Token != "" {
		spec.Token = r.Token
	}
	if r.Endpoint != "" {
		spec.Endpoint = r.Endpoint
	}
	if len(r.KubeConfig) > 0 {
		spec.KubeConfig = r.KubeConfig
	}
	if r.ImageSources != nil {
		spec.ImageSources = convertImageSourcesToCRD(r.ImageSources)
	}
	if r.VeleroInstall != nil {
		spec.VeleroInstall = convertVeleroInstallWriteToCRD(r.VeleroInstall)
	}
}

// PatchDisasterClusterRequest defines the request body for patching a Cluster
type PatchDisasterClusterRequest struct {
	Token         *string                     `json:"token,omitempty"`
	Tag           *string                     `json:"tag,omitempty"`
	Description   *string                     `json:"description,omitempty"`
	ImageSources  *[]ImageSourceDTO           `json:"imageSources,omitempty"`
	VeleroInstall *PatchVeleroInstallWriteDTO `json:"veleroInstall,omitempty"`
}

type RefreshNamespacesRequest struct {
	Type string `json:"type" binding:"required,oneof=namespaceStats workloadNamespaceStats all"`
}

type RefreshNamespacesAcceptedDTO struct {
	Cluster DisasterClusterDTO `json:"cluster"`
	Type    string             `json:"type"`
}

func convertImageSourcesToDTO(items []dapisv1.ImageSource) []ImageSourceDTO {
	if len(items) == 0 {
		return nil
	}
	out := make([]ImageSourceDTO, len(items))
	for i := range items {
		out[i] = ImageSourceDTO{
			Name:     items[i].Name,
			Registry: items[i].Registry,
		}
	}
	return out
}

func convertImageSourcesToCRD(items []ImageSourceDTO) []dapisv1.ImageSource {
	if len(items) == 0 {
		return nil
	}
	out := make([]dapisv1.ImageSource, len(items))
	for i := range items {
		out[i] = dapisv1.ImageSource{
			Name:     items[i].Name,
			Registry: items[i].Registry,
		}
	}
	return out
}

func convertVeleroInstallToDTO(spec *dapisv1.VeleroInstallSpec) *VeleroInstallDTO {
	if spec == nil {
		return nil
	}
	return &VeleroInstallDTO{
		ImageRegistry:        spec.ImageRegistry,
		CredentialConfigured: spec.RegistryCredentialSecretRef != nil && spec.RegistryCredentialSecretRef.Name != "",
	}
}

func convertVeleroInstallWriteToCRD(spec *VeleroInstallWriteDTO) *dapisv1.VeleroInstallSpec {
	if spec == nil {
		return nil
	}
	imageRegistry := strings.Trim(strings.TrimSpace(spec.ImageRegistry), "/")
	if imageRegistry == "" {
		return nil
	}
	return &dapisv1.VeleroInstallSpec{
		ImageRegistry: imageRegistry,
	}
}
