package platformlicenseapi

type StatusDTO struct {
	State              string                 `json:"state"`
	Edition            string                 `json:"edition,omitempty"`
	LicenseID          string                 `json:"licenseId,omitempty"`
	Customer           string                 `json:"customer,omitempty"`
	ExpiresAt          string                 `json:"expiresAt,omitempty"`
	MaxClusters        int                    `json:"maxClusters"`
	ClusterCount       int                    `json:"clusterCount"`
	Fingerprint        string                 `json:"fingerprint"`
	FingerprintVersion string                 `json:"fingerprintVersion"`
	FingerprintRequest *FingerprintRequestDTO `json:"fingerprintRequest,omitempty"`
	FingerprintMatched bool                   `json:"fingerprintMatched"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
	LastCheckedAt      string                 `json:"lastCheckedAt,omitempty"`
	Source             string                 `json:"source"`
	Features           []string               `json:"features,omitempty"`
}

type FingerprintRequestDTO struct {
	Product            string `json:"product"`
	FingerprintVersion string `json:"fingerprintVersion"`
	Fingerprint        string `json:"fingerprint"`
	Namespace          string `json:"namespace"`
	GeneratedAt        string `json:"generatedAt"`
}

type InstallRequest struct {
	License string `json:"license" binding:"required"`
}

type LicenseErrorMeta struct {
	Reason          string `json:"reason"`
	LicenseReason   string `json:"licenseReason,omitempty"`
	State           string `json:"state,omitempty"`
	MaxClusters     int    `json:"maxClusters"`
	CurrentClusters int    `json:"currentClusters"`
}

type ClusterCreateCheck struct {
	Allowed bool
	Status  StatusDTO
	Meta    LicenseErrorMeta
	Message string
}
