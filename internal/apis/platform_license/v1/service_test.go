package platformlicenseapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	platformlicense "github.com/softcdata/testudo-operator/pkg/license"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceCheckClusterCreateIgnoresTamperedStatusConfigMap(t *testing.T) {
	ctx := context.Background()
	client := newRuntimeClient(t,
		&dapisv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"}},
		&dapisv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"}},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: platformlicense.StatusConfigMapName, Namespace: platformlicense.DefaultLicenseNamespace},
			Data: map[string]string{
				"state":              string(platformlicense.StateActive),
				"maxClusters":        "-1",
				"clusterCount":       "2",
				"fingerprintMatched": "true",
			},
		},
	)
	service := &Service{RuntimeClient: client}

	check, err := service.CheckClusterCreate(ctx)

	assert.NoError(t, err)
	assert.False(t, check.Allowed)
	assert.Equal(t, platformlicense.ReasonLicenseLimitExceeded, check.Meta.Reason)
	assert.Equal(t, string(platformlicense.StateFree), check.Meta.State)
	assert.Equal(t, 2, check.Meta.MaxClusters)
	assert.Equal(t, 2, check.Meta.CurrentClusters)
}

func TestServiceStatusUsesConfigMapAndLiveClusterCount(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 16, 10, 20, 30, 0, time.UTC)
	client := newRuntimeClient(t,
		&dapisv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"}},
		&dapisv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster-b"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: metav1.NamespaceSystem, UID: types.UID("kube-system-uid")}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: platformlicense.DefaultLicenseNamespace, UID: types.UID("platform-uid")}},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: platformlicense.StatusConfigMapName, Namespace: platformlicense.DefaultLicenseNamespace},
			Data: map[string]string{
				"state":              string(platformlicense.StateActive),
				"edition":            "enterprise",
				"licenseId":          "LIC-001",
				"customer":           "Acme",
				"expiresAt":          "2027-05-01T00:00:00Z",
				"maxClusters":        "-1",
				"clusterCount":       "99",
				"fingerprintMatched": "true",
				"lastCheckedAt":      "2026-05-09T00:00:00Z",
			},
		},
	)
	service := &Service{
		RuntimeClient: client,
		CABundle:      newTestCABundle(t),
		Now: func() time.Time {
			return now
		},
	}

	status, err := service.Status(ctx)

	assert.NoError(t, err)
	assert.Equal(t, StatusSourceConfigMap, status.Source)
	assert.Equal(t, string(platformlicense.StateActive), status.State)
	assert.Equal(t, -1, status.MaxClusters)
	assert.Equal(t, 2, status.ClusterCount)
	assert.True(t, status.FingerprintMatched)
	assertFingerprintResponse(t, status, platformlicense.DefaultLicenseNamespace, now)
}

func TestServiceStatusReturnsFingerprintWithoutLicense(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 16, 10, 21, 30, 0, time.UTC)
	client := newRuntimeClient(t,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: metav1.NamespaceSystem, UID: types.UID("kube-system-uid")}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: platformlicense.DefaultLicenseNamespace, UID: types.UID("platform-uid")}},
	)
	service := &Service{
		RuntimeClient: client,
		CABundle:      newTestCABundle(t),
		Now: func() time.Time {
			return now
		},
	}

	status, err := service.Status(ctx)

	assert.NoError(t, err)
	assert.Equal(t, StatusSourceLiveEvaluation, status.Source)
	assert.Equal(t, string(platformlicense.StateFree), status.State)
	assert.Equal(t, "NoLicense", status.Reason)
	assert.Equal(t, platformlicense.DefaultFreeMaxClusters, status.MaxClusters)
	assertFingerprintResponse(t, status, platformlicense.DefaultLicenseNamespace, now)

	installID := &corev1.Secret{}
	err = client.Get(ctx, types.NamespacedName{Namespace: platformlicense.DefaultLicenseNamespace, Name: platformlicense.InstallIDSecretName}, installID)
	assert.NoError(t, err)
	assert.NotEmpty(t, installID.Data[platformlicense.InstallIDSecretDataKey])
}

func TestServiceStatusReturnsEnvironmentInvalidWhenFingerprintFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 16, 10, 22, 30, 0, time.UTC)
	client := newRuntimeClient(t)
	service := &Service{
		RuntimeClient: client,
		CAPath:        "/path/that/does/not/exist",
		Now: func() time.Time {
			return now
		},
	}

	status, err := service.Status(ctx)

	assert.NoError(t, err)
	assert.Equal(t, StatusSourceLiveEvaluation, status.Source)
	assert.Equal(t, string(platformlicense.StateUnknown), status.State)
	assert.Equal(t, platformlicense.ReasonLicenseEnvironmentInvalid, status.Reason)
	assert.Equal(t, "", status.Fingerprint)
	assert.Equal(t, platformlicense.FingerprintVersionK8SV1, status.FingerprintVersion)
	assert.Nil(t, status.FingerprintRequest)
	assert.Equal(t, platformlicense.DefaultFreeMaxClusters, status.MaxClusters)
	assert.Equal(t, now.Format(time.RFC3339), status.LastCheckedAt)
	assert.Contains(t, status.Message, "compute deployment fingerprint")
}

func TestServiceStatusPreservesContentErrorWhenFingerprintFails(t *testing.T) {
	ctx := context.Background()
	client := newRuntimeClient(t, platformlicense.BuildLicenseSecret(platformlicense.DefaultLicenseNamespace, []byte(`{}`)))
	service := &Service{
		RuntimeClient: client,
		CAPath:        "/path/that/does/not/exist",
		Now: func() time.Time {
			return time.Date(2026, 5, 16, 10, 23, 30, 0, time.UTC)
		},
	}

	status, err := service.Status(ctx)

	assert.NoError(t, err)
	assert.Equal(t, StatusSourceLiveEvaluation, status.Source)
	assert.Equal(t, string(platformlicense.StateMalformed), status.State)
	assert.Equal(t, platformlicense.ReasonLicenseInvalid, status.Reason)
	assert.Equal(t, "", status.Fingerprint)
	assert.Equal(t, platformlicense.FingerprintVersionK8SV1, status.FingerprintVersion)
	assert.Nil(t, status.FingerprintRequest)
}

func TestServiceInstallWritesLicenseSecret(t *testing.T) {
	ctx := context.Background()
	client := newRuntimeClient(t,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: metav1.NamespaceSystem, UID: types.UID("kube-system-uid")}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: platformlicense.DefaultLicenseNamespace, UID: types.UID("platform-uid")}},
	)
	service := &Service{
		RuntimeClient: client,
		CABundle:      newTestCABundle(t),
		Now: func() time.Time {
			return time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
		},
	}

	status, err := service.Install(ctx, []byte(`{}`))

	assert.NoError(t, err)
	assert.Equal(t, string(platformlicense.StateMalformed), status.State)
	assert.Equal(t, StatusSourceLiveEvaluation, status.Source)
	assertFingerprintResponse(t, status, platformlicense.DefaultLicenseNamespace, time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC))

	secret := &corev1.Secret{}
	err = client.Get(ctx, types.NamespacedName{Namespace: platformlicense.DefaultLicenseNamespace, Name: platformlicense.LicenseSecretName}, secret)
	assert.NoError(t, err)
	assert.Equal(t, corev1.SecretType(platformlicense.LicenseSecretType), secret.Type)
	assert.Equal(t, []byte(`{}`), secret.Data[platformlicense.LicenseSecretDataKey])
}

func TestServiceInstallReturnsCurrentLicenseContentErrorWhenCAPathMissing(t *testing.T) {
	ctx := context.Background()
	client := newRuntimeClient(t,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: platformlicense.StatusConfigMapName, Namespace: platformlicense.DefaultLicenseNamespace},
			Data: map[string]string{
				"state":        string(platformlicense.StateFree),
				"reason":       "NoLicense",
				"maxClusters":  "2",
				"clusterCount": "0",
			},
		},
	)
	service := &Service{
		RuntimeClient: client,
		CAPath:        "/path/that/does/not/exist",
		Now: func() time.Time {
			return time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
		},
	}

	status, err := service.Install(ctx, []byte(`{}`))

	assert.NoError(t, err)
	assert.Equal(t, string(platformlicense.StateMalformed), status.State)
	assert.Equal(t, platformlicense.ReasonLicenseInvalid, status.Reason)
	assert.Equal(t, StatusSourceLiveEvaluation, status.Source)
	assert.Equal(t, "", status.Fingerprint)
	assert.Equal(t, platformlicense.FingerprintVersionK8SV1, status.FingerprintVersion)
	assert.Nil(t, status.FingerprintRequest)
}

func TestServiceInstallReturnsUnknownKeyBeforeFingerprintWhenCAPathMissing(t *testing.T) {
	ctx := context.Background()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)
	client := newRuntimeClient(t)
	service := &Service{
		RuntimeClient: client,
		CAPath:        "/path/that/does/not/exist",
		Now: func() time.Time {
			return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		},
	}

	status, err := service.Install(ctx, signedServerTestLicense(t, privateKey, "unknown-key", func(claims map[string]any) {}))

	assert.NoError(t, err)
	assert.Equal(t, string(platformlicense.StateUnknownKey), status.State)
	assert.Equal(t, platformlicense.ReasonLicenseUnknownKey, status.Reason)
	assert.NotContains(t, status.Message, "read API server CA")
	assert.Equal(t, "", status.Fingerprint)
	assert.Equal(t, platformlicense.FingerprintVersionK8SV1, status.FingerprintVersion)
	assert.Nil(t, status.FingerprintRequest)
}

func TestServiceInstallReturnsEnvironmentReasonForValidLicenseWhenCAPathMissing(t *testing.T) {
	ctx := context.Background()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)
	client := newRuntimeClient(t,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: metav1.NamespaceSystem, UID: types.UID("kube-system-uid")}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: platformlicense.DefaultLicenseNamespace, UID: types.UID("platform-uid")}},
	)
	service := &Service{
		RuntimeClient: client,
		CAPath:        "/path/that/does/not/exist",
		Verifier:      platformlicense.NewVerifier(map[string]ed25519.PublicKey{"test-key": publicKey}),
		Now: func() time.Time {
			return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		},
	}

	status, err := service.Install(ctx, signedServerTestLicense(t, privateKey, "test-key", func(claims map[string]any) {}))

	assert.NoError(t, err)
	assert.Equal(t, string(platformlicense.StateUnknown), status.State)
	assert.Equal(t, platformlicense.ReasonLicenseEnvironmentInvalid, status.Reason)
	assert.Contains(t, status.Message, "compute deployment fingerprint")
	assert.Equal(t, "", status.Fingerprint)
	assert.Equal(t, platformlicense.FingerprintVersionK8SV1, status.FingerprintVersion)
	assert.Nil(t, status.FingerprintRequest)
}

func assertFingerprintResponse(t *testing.T, status StatusDTO, namespace string, generatedAt time.Time) {
	t.Helper()
	assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, status.Fingerprint)
	assert.Equal(t, platformlicense.FingerprintVersionK8SV1, status.FingerprintVersion)
	if assert.NotNil(t, status.FingerprintRequest) {
		assert.Equal(t, platformlicense.ProductName, status.FingerprintRequest.Product)
		assert.Equal(t, platformlicense.FingerprintVersionK8SV1, status.FingerprintRequest.FingerprintVersion)
		assert.Equal(t, status.Fingerprint, status.FingerprintRequest.Fingerprint)
		assert.Equal(t, namespace, status.FingerprintRequest.Namespace)
		assert.Equal(t, generatedAt.Format(time.RFC3339), status.FingerprintRequest.GeneratedAt)
	}
}

func newRuntimeClient(t *testing.T, objects ...runtime.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))
	assert.NoError(t, dapisv1.AddToScheme(scheme))
	return ctrlfake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
}

func newTestCABundle(t *testing.T) []byte {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	assert.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func signedServerTestLicense(t *testing.T, privateKey ed25519.PrivateKey, keyID string, mutate func(map[string]any)) []byte {
	t.Helper()
	claims := map[string]any{
		"version":            1,
		"licenseId":          "LIC-SERVER-TEST-0001",
		"product":            platformlicense.ProductName,
		"customer":           map[string]any{"name": "Acme Corp", "contact": "ops@example.com"},
		"edition":            "enterprise",
		"fingerprintVersion": platformlicense.FingerprintVersionK8SV1,
		"fingerprint":        "sha256:0123456789abcdef",
		"issuedAt":           "2026-05-01T00:00:00Z",
		"notBefore":          "2026-05-01T00:00:00Z",
		"expiresAt":          "2027-05-01T00:00:00Z",
		"limits":             map[string]any{"maxClusters": -1},
		"features":           []any{"cluster.unlimited"},
		"issuer":             "unit-test",
		"keyId":              keyID,
	}
	if mutate != nil {
		mutate(claims)
	}
	unsigned, err := json.Marshal(claims)
	assert.NoError(t, err)
	signature, err := signServerTestRawLicense(unsigned, privateKey)
	assert.NoError(t, err)
	claims["signature"] = signature
	raw, err := json.Marshal(claims)
	assert.NoError(t, err)
	return raw
}

func signServerTestRawLicense(raw []byte, privateKey ed25519.PrivateKey) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("private key must be %d bytes", ed25519.PrivateKeySize)
	}
	payload, err := platformlicense.CanonicalPayload(raw)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, payload)), nil
}
