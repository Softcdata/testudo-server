package userstore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "disaster-system"

func TestEnsureInitializedCreatesAdminSecret(t *testing.T) {
	cli := fake.NewSimpleClientset()
	store := NewKubeStore(cli, testNamespace)

	require.NoError(t, store.EnsureInitialized(context.Background()))

	secret, err := cli.CoreV1().Secrets(testNamespace).Get(context.Background(), UserSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, secret.Data)

	doc := decodeUsersDocFromSecret(t, secret)
	admin, ok := doc.Users[DefaultAdminUser]
	require.True(t, ok)
	assert.Equal(t, DefaultAdminUser, admin.Username)
	assert.Equal(t, DefaultAdminMail, admin.Email)
	assert.Equal(t, RoleAdmin, admin.Role)
	assert.Equal(t, StatusActive, admin.Status)
	assert.NotEmpty(t, admin.PasswordHash)
	assert.GreaterOrEqual(t, doc.NextUserID, int64(2))
}

func TestEnsureInitializedBackfillsMissingAdmin(t *testing.T) {
	raw := mustMarshalUsersDoc(t, &UsersDocument{
		SchemaVersion: SchemaVersion,
		NextUserID:    9,
		Users: map[string]UserRecord{
			"ops": {
				ID:           8,
				Username:     "ops",
				Email:        "ops@example.com",
				Role:         RoleAdmin,
				Status:       StatusActive,
				PasswordHash: "hash",
				CreatedAt:    "2026-04-01T00:00:00Z",
				UpdatedAt:    "2026-04-01T00:00:00Z",
			},
		},
	})

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: UserSecretName, Namespace: testNamespace},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{UserSecretKey: raw},
	}

	cli := fake.NewSimpleClientset(secret)
	store := NewKubeStore(cli, testNamespace)

	require.NoError(t, store.EnsureInitialized(context.Background()))

	updated, err := cli.CoreV1().Secrets(testNamespace).Get(context.Background(), UserSecretName, metav1.GetOptions{})
	require.NoError(t, err)
	doc := decodeUsersDocFromSecret(t, updated)
	_, ok := doc.Users[DefaultAdminUser]
	require.True(t, ok)
	assert.GreaterOrEqual(t, doc.NextUserID, int64(10))
}

func TestCreateUserDuplicateAndStatusUpdate(t *testing.T) {
	cli := fake.NewSimpleClientset()
	store := NewKubeStore(cli, testNamespace)

	require.NoError(t, store.EnsureInitialized(context.Background()))

	created, err := store.CreateUser(context.Background(), CreateUserInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "test-password",
		Actor:    "admin",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice", created.Username)
	assert.Equal(t, "alice@example.com", created.Email)
	assert.Equal(t, RoleAdmin, created.Role)
	assert.Equal(t, StatusActive, created.Status)
	assert.NotEmpty(t, created.PasswordHash)

	_, err = store.CreateUser(context.Background(), CreateUserInput{
		Username: "alice",
		Email:    "alice2@example.com",
		Password: "test-password",
	})
	require.ErrorIs(t, err, ErrUserNameExists)

	_, err = store.CreateUser(context.Background(), CreateUserInput{
		Username: "alice2",
		Email:    "alice@example.com",
		Password: "test-password",
	})
	require.ErrorIs(t, err, ErrUserEmailExists)

	updated, err := store.UpdateUserStatus(context.Background(), "alice", StatusDisabled, "admin")
	require.NoError(t, err)
	assert.Equal(t, StatusDisabled, updated.Status)

	fetched, err := store.GetUserByUsername(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, StatusDisabled, fetched.Status)

	_, err = store.UpdateUserStatus(context.Background(), "alice", "paused", "admin")
	require.ErrorIs(t, err, ErrInvalidUserStatus)
}

func TestListUsersDeleteAndUpdatePassword(t *testing.T) {
	cli := fake.NewSimpleClientset()
	store := NewKubeStore(cli, testNamespace)

	require.NoError(t, store.EnsureInitialized(context.Background()))
	_, err := store.CreateUser(context.Background(), CreateUserInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "test-password",
		Actor:    "admin",
	})
	require.NoError(t, err)

	users, err := store.ListUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, "admin", users[0].Username)
	assert.Equal(t, "alice", users[1].Username)

	_, err = store.UpdateUserPassword(context.Background(), "alice", "new-password", "admin")
	require.NoError(t, err)

	fetched, err := store.GetUserByUsername(context.Background(), "alice")
	require.NoError(t, err)
	require.NoError(t, VerifyPassword(fetched.PasswordHash, "new-password"))
	require.Error(t, VerifyPassword(fetched.PasswordHash, "old-password"))

	require.NoError(t, store.DeleteUser(context.Background(), "alice", "admin"))
	_, err = store.GetUserByUsername(context.Background(), "alice")
	require.ErrorIs(t, err, ErrUserNotFound)

	err = store.DeleteUser(context.Background(), DefaultAdminUser, "admin")
	require.ErrorIs(t, err, ErrDeleteBuiltInUser)

	err = store.DeleteUser(context.Background(), "ghost", "admin")
	require.ErrorIs(t, err, ErrUserNotFound)
}

func decodeUsersDocFromSecret(t *testing.T, secret *corev1.Secret) *UsersDocument {
	t.Helper()
	var doc UsersDocument
	err := json.Unmarshal(secret.Data[UserSecretKey], &doc)
	require.NoError(t, err)
	if doc.Users == nil {
		doc.Users = map[string]UserRecord{}
	}
	return &doc
}

func mustMarshalUsersDoc(t *testing.T, doc *UsersDocument) []byte {
	t.Helper()
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	return raw
}
