package systemsettings

type SystemSettingItem struct {
	Name      string `json:"name"`
	ConfigKey string `json:"config_key"`
	Value     string `json:"value"`
	Remark    string `json:"remark"`
}

type CreateSystemSettingRequest struct {
	Name      string `json:"name"`
	ConfigKey string `json:"config_key"`
	Value     string `json:"value"`
	Remark    string `json:"remark"`
}

type UpdateSystemSettingRequest struct {
	Name   *string `json:"name,omitempty"`
	Value  *string `json:"value,omitempty"`
	Remark *string `json:"remark,omitempty"`
}

type settingsDocument struct {
	SchemaVersion int                          `json:"schemaVersion"`
	Items         map[string]SystemSettingItem `json:"items"`
	UpdatedAt     string                       `json:"updatedAt,omitempty"`
	UpdatedBy     string                       `json:"updatedBy,omitempty"`
}
