package businessdefaultconfig

type FieldValueType string

const (
	FieldValueTypeDuration FieldValueType = "duration"
	FieldValueTypeInt      FieldValueType = "int"
	FieldValueTypeBool     FieldValueType = "bool"
	FieldValueTypeString   FieldValueType = "string"
	FieldValueTypeEnum     FieldValueType = "enum"
	FieldValueTypeObject   FieldValueType = "object"
)

type EffectMode string

const (
	EffectModeHot     EffectMode = "hot"
	EffectModeRestart EffectMode = "restart"
	EffectModeStartup EffectMode = "startup"
)

type FieldDTO struct {
	Key          string         `json:"key"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Value        interface{}    `json:"value"`
	DefaultValue interface{}    `json:"defaultValue"`
	DataType     FieldValueType `json:"dataType"`
	Editable     bool           `json:"editable"`
	EffectMode   EffectMode     `json:"effectMode"`
	Min          interface{}    `json:"min,omitempty"`
	Max          interface{}    `json:"max,omitempty"`
	EnumValues   []string       `json:"enumValues,omitempty"`
	GroupKey     string         `json:"groupKey"`
	GroupName    string         `json:"groupName"`
}

type FrontendSpecFieldDTO struct {
	Key                 string                    `json:"key"`
	Value               interface{}               `json:"value"`
	Name                string                    `json:"name"`
	Description         string                    `json:"description"`
	ResourceKind        string                    `json:"resourceKind"`
	RequestPath         string                    `json:"requestPath"`
	SpecPath            string                    `json:"specPath"`
	ConfigGroupKey      string                    `json:"configGroupKey"`
	ConfigKey           string                    `json:"configKey"`
	DataType            FieldValueType            `json:"dataType"`
	Editable            bool                      `json:"editable"`
	ServerSupported     bool                      `json:"serverSupported"`
	SupportedOperations []string                  `json:"supportedOperations,omitempty"`
	KeySegments         []string                  `json:"keySegments"`
	RequestPathSegments []string                  `json:"requestPathSegments"`
	SpecPathSegments    []string                  `json:"specPathSegments"`
	APIUsages           []FrontendSpecAPIUsageDTO `json:"apiUsages"`
	Note                string                    `json:"note,omitempty"`
}

type FrontendSpecAPIUsageDTO struct {
	Method              string   `json:"method"`
	Path                string   `json:"path"`
	RequestPath         string   `json:"requestPath,omitempty"`
	RequestPathSegments []string `json:"requestPathSegments,omitempty"`
	ResourceKind        string   `json:"resourceKind,omitempty"`
	Operation           string   `json:"operation,omitempty"`
	Description         string   `json:"description,omitempty"`
}

type FrontendSpecFieldCollectionDTO struct {
	Items    []FrontendSpecFieldDTO          `json:"items"`
	FieldMap map[string]FrontendSpecFieldDTO `json:"fieldMap"`
}

type GroupDTO struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Fields      []FieldDTO `json:"fields"`
}

type SnapshotDTO struct {
	SchemaVersion int        `json:"schemaVersion"`
	UpdatedAt     string     `json:"updatedAt,omitempty"`
	UpdatedBy     string     `json:"updatedBy,omitempty"`
	Groups        []GroupDTO `json:"groups"`
}

type FieldErrorMeta struct {
	Field       string      `json:"field,omitempty"`
	Reason      string      `json:"reason"`
	Expected    string      `json:"expected,omitempty"`
	Min         interface{} `json:"min,omitempty"`
	Max         interface{} `json:"max,omitempty"`
	EnumValues  []string    `json:"enumValues,omitempty"`
	ActualValue interface{} `json:"actualValue,omitempty"`
}

type configDocument struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Values        map[string]interface{} `json:"values"`
	UpdatedAt     string                 `json:"updatedAt,omitempty"`
	UpdatedBy     string                 `json:"updatedBy,omitempty"`
}
