package businessdefaultconfig

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type requestValidationError struct {
	meta FieldErrorMeta
}

func (e *requestValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.meta.Field != "" {
		return fmt.Sprintf("%s: %s", e.meta.Field, e.meta.Reason)
	}
	return e.meta.Reason
}

func (e *requestValidationError) Meta() FieldErrorMeta {
	if e == nil {
		return FieldErrorMeta{}
	}
	return e.meta
}

func newValidationError(field fieldDefinition, reason string, actual interface{}) *requestValidationError {
	return &requestValidationError{
		meta: FieldErrorMeta{
			Field:       field.Key,
			Reason:      reason,
			Expected:    string(field.DataType),
			Min:         field.Min,
			Max:         field.Max,
			EnumValues:  append([]string(nil), field.EnumValues...),
			ActualValue: actual,
		},
	}
}

func normalizeDocumentValues(doc *configDocument) error {
	if doc == nil {
		return nil
	}
	if doc.Values == nil {
		doc.Values = make(map[string]interface{})
		return nil
	}

	normalized := make(map[string]interface{}, len(doc.Values))
	for key, value := range doc.Values {
		field, ok := fieldDefinitionByKey(key)
		if !ok {
			continue
		}
		normalizedValue, err := validateAndNormalizeValue(field, value)
		if err != nil {
			return err
		}
		normalized[key] = normalizedValue
	}
	doc.Values = normalized
	return validateCrossFieldValues(effectiveValues(doc))
}

func validateAndNormalizeValue(field fieldDefinition, raw interface{}) (interface{}, error) {
	if raw == nil {
		return nil, newValidationError(field, "value is required", nil)
	}

	switch field.DataType {
	case FieldValueTypeDuration:
		value, ok := raw.(string)
		if !ok {
			return nil, newValidationError(field, "duration value must be a string with unit, for example 15s or 5m", raw)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, newValidationError(field, "duration value is required", raw)
		}
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return nil, newValidationError(field, "duration value cannot be parsed", raw)
		}
		min, max, err := durationRange(field)
		if err != nil {
			return nil, err
		}
		if parsed < min {
			return nil, newValidationError(field, "duration value is below minimum", value)
		}
		if parsed > max {
			return nil, newValidationError(field, "duration value is above maximum", value)
		}
		return value, nil
	case FieldValueTypeInt:
		value, ok := parseInt(raw)
		if !ok {
			return nil, newValidationError(field, "integer value must be a JSON number", raw)
		}
		min, max, err := intRange(field)
		if err != nil {
			return nil, err
		}
		if value < min {
			return nil, newValidationError(field, "integer value is below minimum", value)
		}
		if value > max {
			return nil, newValidationError(field, "integer value is above maximum", value)
		}
		return int(value), nil
	case FieldValueTypeBool:
		value, ok := raw.(bool)
		if !ok {
			return nil, newValidationError(field, "boolean value must be true or false", raw)
		}
		return value, nil
	case FieldValueTypeString:
		value, ok := raw.(string)
		if !ok {
			return nil, newValidationError(field, "string value is required", raw)
		}
		return value, nil
	case FieldValueTypeEnum:
		value, ok := raw.(string)
		if !ok {
			return nil, newValidationError(field, "enum value must be a string", raw)
		}
		for _, allowed := range field.EnumValues {
			if value == allowed {
				return value, nil
			}
		}
		return nil, newValidationError(field, "enum value is not allowed", raw)
	default:
		return nil, newValidationError(field, "unsupported field data type", raw)
	}
}

func durationRange(field fieldDefinition) (time.Duration, time.Duration, error) {
	min, ok := field.Min.(string)
	if !ok {
		return 0, 0, fmt.Errorf("%s: invalid duration minimum metadata", field.Key)
	}
	max, ok := field.Max.(string)
	if !ok {
		return 0, 0, fmt.Errorf("%s: invalid duration maximum metadata", field.Key)
	}
	minDuration, err := time.ParseDuration(min)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: invalid duration minimum metadata: %w", field.Key, err)
	}
	maxDuration, err := time.ParseDuration(max)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: invalid duration maximum metadata: %w", field.Key, err)
	}
	return minDuration, maxDuration, nil
}

func intRange(field fieldDefinition) (int64, int64, error) {
	min, ok := parseInt(field.Min)
	if !ok {
		return 0, 0, fmt.Errorf("%s: invalid integer minimum metadata", field.Key)
	}
	max, ok := parseInt(field.Max)
	if !ok {
		return 0, 0, fmt.Errorf("%s: invalid integer maximum metadata", field.Key)
	}
	return min, max, nil
}

func parseInt(raw interface{}) (int64, bool) {
	switch value := raw.(type) {
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case float64:
		if math.Trunc(value) != value {
			return 0, false
		}
		return int64(value), true
	case json.Number:
		parsed, err := strconv.ParseInt(value.String(), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func effectiveValues(doc *configDocument) map[string]interface{} {
	values := make(map[string]interface{}, len(fieldDefinitions))
	for _, field := range fieldDefinitions {
		value, exists := doc.Values[field.Key]
		if !exists {
			value = field.DefaultValue
		}
		values[field.Key] = normalizeStoredValueForDTO(field, value)
	}
	return values
}

func normalizeStoredValueForDTO(field fieldDefinition, value interface{}) interface{} {
	switch field.DataType {
	case FieldValueTypeInt:
		if parsed, ok := parseInt(value); ok {
			return int(parsed)
		}
	case FieldValueTypeDuration:
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return value
}

func validateCrossFieldValues(values map[string]interface{}) error {
	transition, err := durationValue(values, "instanceRuntime.transitionWatchdogTimeout")
	if err != nil {
		return err
	}
	minTransition, err := durationValue(values, "instanceRuntime.minTransitionWatchdogTimeout")
	if err != nil {
		return err
	}
	if transition < minTransition {
		field := fieldsByKey["instanceRuntime.transitionWatchdogTimeout"]
		return &requestValidationError{
			meta: FieldErrorMeta{
				Field:       field.Key,
				Reason:      "transitionWatchdogTimeout must be greater than or equal to minTransitionWatchdogTimeout",
				Expected:    string(field.DataType),
				Min:         values["instanceRuntime.minTransitionWatchdogTimeout"],
				Max:         field.Max,
				ActualValue: values["instanceRuntime.transitionWatchdogTimeout"],
			},
		}
	}
	return nil
}

func durationValue(values map[string]interface{}, key string) (time.Duration, error) {
	value, ok := values[key]
	if !ok {
		return 0, fmt.Errorf("%s: value is missing", key)
	}
	text, ok := value.(string)
	if !ok {
		return 0, fmt.Errorf("%s: duration value must be a string", key)
	}
	parsed, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("%s: duration value cannot be parsed: %w", key, err)
	}
	return parsed, nil
}

func actorFromContextValue(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "system"
	}
	return actor
}
