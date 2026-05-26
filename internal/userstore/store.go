package userstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const (
	UserSecretName   = "disaster-server-users"
	UserSecretKey    = "users.json"
	SchemaVersion    = 1
	RoleAdmin        = "admin"
	StatusActive     = "active"
	StatusDisabled   = "disabled"
	DefaultAdminUser = "admin"
	DefaultAdminPass = "123456"
	DefaultAdminMail = "admin@example.com"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserNameExists      = errors.New("username already exists")
	ErrUserEmailExists     = errors.New("email already exists")
	ErrInvalidUserStatus   = errors.New("invalid user status")
	ErrDeleteBuiltInUser   = errors.New("built-in admin user cannot be deleted")
	ErrStoreNotReady       = errors.New("user store is not initialized")
	ErrInvalidSecretDoc    = errors.New("invalid user secret document")
	ErrMissingPasswordHash = errors.New("missing password hash")
)

type UserRecord struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	PasswordHash string `json:"passwordHash"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type UsersDocument struct {
	SchemaVersion int64                 `json:"schemaVersion"`
	NextUserID    int64                 `json:"nextUserID"`
	UpdatedAt     string                `json:"updatedAt,omitempty"`
	UpdatedBy     string                `json:"updatedBy,omitempty"`
	Users         map[string]UserRecord `json:"users"`
}

type CreateUserInput struct {
	Username string
	Email    string
	Password string
	Actor    string
}

type Store interface {
	EnsureInitialized(ctx context.Context) error
	GetUserByUsername(ctx context.Context, username string) (UserRecord, error)
	ListUsers(ctx context.Context) ([]UserRecord, error)
	CreateUser(ctx context.Context, input CreateUserInput) (UserRecord, error)
	DeleteUser(ctx context.Context, username, actor string) error
	UpdateUserPassword(ctx context.Context, username, password, actor string) (UserRecord, error)
	UpdateUserStatus(ctx context.Context, username, status, actor string) (UserRecord, error)
}

type KubeStore struct {
	client    kubernetes.Interface
	namespace string
}

func NewKubeStore(client kubernetes.Interface, namespace string) *KubeStore {
	return &KubeStore{client: client, namespace: namespace}
}

func (s *KubeStore) EnsureInitialized(ctx context.Context) error {
	if s == nil || s.client == nil {
		return ErrStoreNotReady
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, UserSecretName, metav1.GetOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}

			doc, err := newDefaultDocument()
			if err != nil {
				return err
			}
			raw, err := encodeDocument(doc)
			if err != nil {
				return err
			}

			newSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      UserSecretName,
					Namespace: s.namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{UserSecretKey: raw},
			}
			if _, createErr := s.client.CoreV1().Secrets(s.namespace).Create(ctx, newSecret, metav1.CreateOptions{}); createErr != nil {
				if k8serrors.IsAlreadyExists(createErr) {
					return k8serrors.NewConflict(schema.GroupResource{Resource: "secrets"}, UserSecretName, createErr)
				}
				return createErr
			}
			return nil
		}

		doc, err := decodeDocument(secret)
		if err != nil {
			return err
		}

		if _, ok := doc.Users[DefaultAdminUser]; ok {
			return nil
		}

		admin, err := newAdminRecord(doc.NextUserID)
		if err != nil {
			return err
		}
		doc.Users[DefaultAdminUser] = admin
		if doc.NextUserID <= admin.ID {
			doc.NextUserID = admin.ID + 1
		}
		now := time.Now().UTC().Format(time.RFC3339)
		doc.UpdatedAt = now
		doc.UpdatedBy = "system"

		raw, err := encodeDocument(doc)
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[UserSecretKey] = raw
		_, err = s.client.CoreV1().Secrets(s.namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	})
}

func (s *KubeStore) GetUserByUsername(ctx context.Context, username string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, ErrStoreNotReady
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return UserRecord{}, ErrUserNotFound
	}

	secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, UserSecretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return UserRecord{}, ErrUserNotFound
		}
		return UserRecord{}, err
	}

	doc, err := decodeDocument(secret)
	if err != nil {
		return UserRecord{}, err
	}

	user, ok := doc.Users[username]
	if !ok {
		return UserRecord{}, ErrUserNotFound
	}
	user = normalizeUser(username, user)
	if user.PasswordHash == "" {
		return UserRecord{}, ErrMissingPasswordHash
	}
	return user, nil
}

func (s *KubeStore) ListUsers(ctx context.Context) ([]UserRecord, error) {
	if s == nil || s.client == nil {
		return nil, ErrStoreNotReady
	}

	if err := s.EnsureInitialized(ctx); err != nil {
		return nil, err
	}

	secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, UserSecretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return []UserRecord{}, nil
		}
		return nil, err
	}

	doc, err := decodeDocument(secret)
	if err != nil {
		return nil, err
	}

	users := make([]UserRecord, 0, len(doc.Users))
	for key, user := range doc.Users {
		users = append(users, normalizeUser(key, user))
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})

	return users, nil
}

func (s *KubeStore) CreateUser(ctx context.Context, input CreateUserInput) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, ErrStoreNotReady
	}

	if err := s.EnsureInitialized(ctx); err != nil {
		return UserRecord{}, err
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.Actor = strings.TrimSpace(input.Actor)
	if input.Actor == "" {
		input.Actor = "system"
	}

	var created UserRecord
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, UserSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		doc, err := decodeDocument(secret)
		if err != nil {
			return err
		}

		if _, exists := doc.Users[input.Username]; exists {
			return ErrUserNameExists
		}

		for _, user := range doc.Users {
			if strings.EqualFold(strings.TrimSpace(user.Email), input.Email) {
				return ErrUserEmailExists
			}
		}

		hash, err := HashPassword(input.Password)
		if err != nil {
			return err
		}

		now := time.Now().UTC().Format(time.RFC3339)
		id := nextUserID(doc)
		created = UserRecord{
			ID:           id,
			Username:     input.Username,
			Email:        input.Email,
			Role:         RoleAdmin,
			Status:       StatusActive,
			PasswordHash: hash,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		doc.Users[input.Username] = created
		doc.NextUserID = id + 1
		doc.UpdatedAt = now
		doc.UpdatedBy = input.Actor

		raw, err := encodeDocument(doc)
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[UserSecretKey] = raw
		_, err = s.client.CoreV1().Secrets(s.namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return UserRecord{}, err
	}

	return normalizeUser(created.Username, created), nil
}

func (s *KubeStore) DeleteUser(ctx context.Context, username, actor string) error {
	if s == nil || s.client == nil {
		return ErrStoreNotReady
	}

	if err := s.EnsureInitialized(ctx); err != nil {
		return err
	}

	username = strings.TrimSpace(username)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	if username == "" {
		return ErrUserNotFound
	}
	if username == DefaultAdminUser {
		return ErrDeleteBuiltInUser
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, UserSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		doc, err := decodeDocument(secret)
		if err != nil {
			return err
		}
		if _, exists := doc.Users[username]; !exists {
			return ErrUserNotFound
		}

		delete(doc.Users, username)
		now := time.Now().UTC().Format(time.RFC3339)
		doc.UpdatedAt = now
		doc.UpdatedBy = actor

		raw, err := encodeDocument(doc)
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[UserSecretKey] = raw
		_, err = s.client.CoreV1().Secrets(s.namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *KubeStore) UpdateUserPassword(ctx context.Context, username, password, actor string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, ErrStoreNotReady
	}

	if err := s.EnsureInitialized(ctx); err != nil {
		return UserRecord{}, err
	}

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	if username == "" {
		return UserRecord{}, ErrUserNotFound
	}
	if password == "" {
		return UserRecord{}, ErrEmptyPassword
	}

	hash, err := HashPassword(password)
	if err != nil {
		return UserRecord{}, err
	}

	var updated UserRecord
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, UserSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		doc, err := decodeDocument(secret)
		if err != nil {
			return err
		}

		user, exists := doc.Users[username]
		if !exists {
			return ErrUserNotFound
		}

		now := time.Now().UTC().Format(time.RFC3339)
		user = normalizeUser(username, user)
		user.PasswordHash = hash
		user.UpdatedAt = now
		doc.Users[username] = user
		doc.UpdatedAt = now
		doc.UpdatedBy = actor
		updated = user

		raw, err := encodeDocument(doc)
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[UserSecretKey] = raw
		_, err = s.client.CoreV1().Secrets(s.namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return UserRecord{}, ErrUserNotFound
		}
		return UserRecord{}, err
	}

	return updated, nil
}

func (s *KubeStore) UpdateUserStatus(ctx context.Context, username, status, actor string) (UserRecord, error) {
	if s == nil || s.client == nil {
		return UserRecord{}, ErrStoreNotReady
	}

	if err := s.EnsureInitialized(ctx); err != nil {
		return UserRecord{}, err
	}

	username = strings.TrimSpace(username)
	status = strings.TrimSpace(status)
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	if status != StatusActive && status != StatusDisabled {
		return UserRecord{}, ErrInvalidUserStatus
	}

	var updated UserRecord
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, UserSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		doc, err := decodeDocument(secret)
		if err != nil {
			return err
		}

		user, exists := doc.Users[username]
		if !exists {
			return ErrUserNotFound
		}

		now := time.Now().UTC().Format(time.RFC3339)
		user = normalizeUser(username, user)
		user.Status = status
		user.UpdatedAt = now
		doc.Users[username] = user
		doc.UpdatedAt = now
		doc.UpdatedBy = actor
		updated = user

		raw, err := encodeDocument(doc)
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[UserSecretKey] = raw
		_, err = s.client.CoreV1().Secrets(s.namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return UserRecord{}, ErrUserNotFound
		}
		return UserRecord{}, err
	}

	return updated, nil
}

func newDefaultDocument() (*UsersDocument, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	admin, err := newAdminRecord(1)
	if err != nil {
		return nil, err
	}
	admin.CreatedAt = now
	admin.UpdatedAt = now

	return &UsersDocument{
		SchemaVersion: SchemaVersion,
		NextUserID:    2,
		UpdatedAt:     now,
		UpdatedBy:     "system",
		Users: map[string]UserRecord{
			DefaultAdminUser: admin,
		},
	}, nil
}

func newAdminRecord(id int64) (UserRecord, error) {
	hash, err := HashPassword(DefaultAdminPass)
	if err != nil {
		return UserRecord{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return UserRecord{
		ID:           id,
		Username:     DefaultAdminUser,
		Email:        DefaultAdminMail,
		Role:         RoleAdmin,
		Status:       StatusActive,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func decodeDocument(secret *corev1.Secret) (*UsersDocument, error) {
	if secret == nil {
		return nil, ErrInvalidSecretDoc
	}

	raw := []byte{}
	if secret.Data != nil {
		raw = secret.Data[UserSecretKey]
	}

	doc := &UsersDocument{}
	if len(strings.TrimSpace(string(raw))) == 0 {
		doc.SchemaVersion = SchemaVersion
		doc.Users = map[string]UserRecord{}
		doc.NextUserID = 1
		return doc, nil
	}

	if err := json.Unmarshal(raw, doc); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSecretDoc, err)
	}

	if doc.Users == nil {
		doc.Users = map[string]UserRecord{}
	}
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = SchemaVersion
	}

	maxID := int64(0)
	for key, user := range doc.Users {
		normalized := normalizeUser(key, user)
		doc.Users[key] = normalized
		if normalized.ID > maxID {
			maxID = normalized.ID
		}
	}
	if doc.NextUserID <= maxID {
		doc.NextUserID = maxID + 1
	}
	if doc.NextUserID <= 0 {
		doc.NextUserID = 1
	}

	return doc, nil
}

func encodeDocument(doc *UsersDocument) ([]byte, error) {
	if doc == nil {
		return nil, ErrInvalidSecretDoc
	}
	if doc.Users == nil {
		doc.Users = map[string]UserRecord{}
	}
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = SchemaVersion
	}
	if doc.NextUserID <= 0 {
		doc.NextUserID = nextUserID(doc)
	}

	return json.Marshal(doc)
}

func normalizeUser(key string, user UserRecord) UserRecord {
	if user.Username == "" {
		user.Username = key
	}
	if user.Role == "" {
		user.Role = RoleAdmin
	}
	if user.Status == "" {
		user.Status = StatusActive
	}
	if user.Username == DefaultAdminUser && strings.TrimSpace(user.Email) == "" {
		user.Email = DefaultAdminMail
	}
	return user
}

func nextUserID(doc *UsersDocument) int64 {
	if doc.NextUserID > 0 {
		return doc.NextUserID
	}
	maxID := int64(0)
	for _, user := range doc.Users {
		if user.ID > maxID {
			maxID = user.ID
		}
	}
	return maxID + 1
}
