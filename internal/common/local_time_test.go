package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLocalTimeMarshalJSONUsesProcessLocalTimezone(t *testing.T) {
	originalLocal := time.Local
	t.Cleanup(func() {
		time.Local = originalLocal
	})
	time.Local = time.FixedZone("CST", 8*60*60)

	value := NewLocalTime(metav1.NewTime(time.Date(2026, 5, 15, 3, 8, 19, 0, time.UTC)))

	data, err := json.Marshal(value)
	require.NoError(t, err)
	assert.JSONEq(t, `"2026-05-15T11:08:19+08:00"`, string(data))
}

func TestLocalTimeUnmarshalJSON(t *testing.T) {
	var value LocalTime

	require.NoError(t, json.Unmarshal([]byte(`"2026-05-15T11:08:19+08:00"`), &value))

	assert.True(t, value.Time.Time.UTC().Equal(time.Date(2026, 5, 15, 3, 8, 19, 0, time.UTC)))
}
