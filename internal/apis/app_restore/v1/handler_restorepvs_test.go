package apprestore

import (
	"testing"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/stretchr/testify/assert"
)

func TestNeedsCleanVolumeRuleForCreate(t *testing.T) {
	restorePVsTrue := true
	restorePVsFalse := false

	assert.True(t, needsCleanVolumeRuleForCreate(CreateAppRestoreRequest{
		CleanVolumes: false,
		RestorePVs:   &restorePVsTrue,
	}))

	assert.True(t, needsCleanVolumeRuleForCreate(CreateAppRestoreRequest{
		CleanVolumes: true,
		RestorePVs:   &restorePVsFalse,
	}))

	assert.False(t, needsCleanVolumeRuleForCreate(CreateAppRestoreRequest{
		CleanVolumes: false,
		RestorePVs:   &restorePVsFalse,
	}))
}

func TestNeedsCleanVolumeRuleForUpdate(t *testing.T) {
	restorePVsTrue := true
	restorePVsFalse := false
	cleanTrue := true
	cleanFalse := false

	assert.True(t, needsCleanVolumeRuleForUpdate(UpdateAppRestoreRequest{
		CleanVolumes: &cleanFalse,
		RestorePVs:   &restorePVsTrue,
	}))

	assert.True(t, needsCleanVolumeRuleForUpdate(UpdateAppRestoreRequest{
		CleanVolumes: &cleanTrue,
		RestorePVs:   &restorePVsFalse,
	}))

	assert.False(t, needsCleanVolumeRuleForUpdate(UpdateAppRestoreRequest{
		CleanVolumes: &cleanFalse,
		RestorePVs:   &restorePVsFalse,
	}))
}

func TestEnsureCleanVolumeRule_NoDuplicate(t *testing.T) {
	rules := []dapisv1.ResourceModifierRule{
		{
			Conditions: dapisv1.Conditions{
				GroupResource: "persistentvolumeclaims",
			},
			Patches: []dapisv1.JSONPatch{
				{
					Operation: "add",
					Path:      "/spec/volumeName",
					Value:     "",
				},
			},
		},
	}

	got := ensureCleanVolumeRule(rules)
	assert.Len(t, got, 1)
	assert.True(t, hasCleanVolumeRule(got))
	assert.Equal(t, "add", got[0].Patches[0].Operation)
	assert.Equal(t, "", got[0].Patches[0].Value)
}

func TestEnsureCleanVolumeRule_ReplacesLegacyRemove(t *testing.T) {
	rules := []dapisv1.ResourceModifierRule{
		{
			Conditions: dapisv1.Conditions{
				GroupResource: "persistentvolumeclaims",
			},
			Patches: []dapisv1.JSONPatch{
				{
					Operation: "remove",
					Path:      "/spec/volumeName",
				},
			},
		},
	}

	got := ensureCleanVolumeRule(rules)
	assert.Len(t, got, 1)
	assert.True(t, hasCleanVolumeRule(got))
	assert.Equal(t, "add", got[0].Patches[0].Operation)
	assert.Equal(t, "/spec/volumeName", got[0].Patches[0].Path)
	assert.Equal(t, "", got[0].Patches[0].Value)
}

func TestEnsureCleanVolumeRule_AutoAppend(t *testing.T) {
	rules := []dapisv1.ResourceModifierRule{
		{
			Conditions: dapisv1.Conditions{
				GroupResource: "ingresses.networking.k8s.io",
			},
			Patches: []dapisv1.JSONPatch{
				{
					Operation: "replace",
					Path:      "/spec/ingressClassName",
					Value:     "traefik",
				},
			},
		},
	}

	got := ensureCleanVolumeRule(rules)
	assert.Len(t, got, 2)
	assert.True(t, hasCleanVolumeRule(got))
	assert.Equal(t, "add", got[1].Patches[0].Operation)
	assert.Equal(t, "", got[1].Patches[0].Value)
}
