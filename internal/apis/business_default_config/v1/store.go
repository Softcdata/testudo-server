package businessdefaultconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/softcdata/testudo-server/internal/common"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const (
	configMapName       = "disaster-business-default-config"
	configDataKey       = "config"
	configSchemaVersion = 1
	maxConfigJSONSize   = 256 * 1024
)

var errConfigTooLarge = errors.New("business default config payload exceeds limit")

type configMapStore interface {
	Get(ctx context.Context, name string) (*corev1.ConfigMap, error)
	Create(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	Update(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
}

type kubeConfigMapStore struct {
	client kubernetes.Interface
}

func (s *kubeConfigMapStore) Get(ctx context.Context, name string) (*corev1.ConfigMap, error) {
	return s.client.CoreV1().ConfigMaps(common.DisasterSystemNamespace).Get(ctx, name, metav1.GetOptions{})
}

func (s *kubeConfigMapStore) Create(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return s.client.CoreV1().ConfigMaps(common.DisasterSystemNamespace).Create(ctx, configMap, metav1.CreateOptions{})
}

func (s *kubeConfigMapStore) Update(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return s.client.CoreV1().ConfigMaps(common.DisasterSystemNamespace).Update(ctx, configMap, metav1.UpdateOptions{})
}

func defaultConfigDocument() *configDocument {
	return &configDocument{
		SchemaVersion: configSchemaVersion,
		Values:        make(map[string]interface{}),
		UpdatedBy:     "system",
	}
}

func decodeConfigDocument(cm *corev1.ConfigMap) (*configDocument, error) {
	doc := defaultConfigDocument()
	if cm == nil || cm.Data == nil {
		return doc, nil
	}

	raw := strings.TrimSpace(cm.Data[configDataKey])
	if raw == "" {
		return doc, nil
	}

	if err := json.Unmarshal([]byte(raw), doc); err != nil {
		return nil, fmt.Errorf("invalid business default config json: %w", err)
	}
	if doc.SchemaVersion == 0 {
		doc.SchemaVersion = configSchemaVersion
	}
	if doc.Values == nil {
		doc.Values = make(map[string]interface{})
	}
	if doc.UpdatedBy == "" {
		doc.UpdatedBy = "system"
	}
	if err := normalizeDocumentValues(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (h *Handler) getConfigDocument(c context.Context) (*configDocument, error) {
	if h.store == nil {
		return nil, errors.New("business default config store is not initialized")
	}

	cm, err := h.store.Get(c, configMapName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return defaultConfigDocument(), nil
		}
		return nil, err
	}
	return decodeConfigDocument(cm)
}

func (h *Handler) mutateConfigDocument(c context.Context, actor string, mutate func(doc *configDocument) error) (*configDocument, error) {
	if h.store == nil {
		return nil, errors.New("business default config store is not initialized")
	}

	var result *configDocument
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, doc, err := h.loadConfigMap(c)
		if err != nil {
			return err
		}

		if err := mutate(doc); err != nil {
			return err
		}
		if err := normalizeDocumentValues(doc); err != nil {
			return err
		}

		doc.SchemaVersion = configSchemaVersion
		doc.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		doc.UpdatedBy = actorFromContextValue(actor)

		payload, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		if len(payload) > maxConfigJSONSize {
			return errConfigTooLarge
		}

		if cm == nil {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: common.DisasterSystemNamespace,
				},
				Data: map[string]string{
					configDataKey: string(payload),
				},
			}
			if _, err := h.store.Create(c, cm); err != nil {
				return err
			}
		} else {
			if cm.Data == nil {
				cm.Data = make(map[string]string)
			}
			cm.Data[configDataKey] = string(payload)
			if _, err := h.store.Update(c, cm); err != nil {
				return err
			}
		}

		result = doc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Handler) loadConfigMap(c context.Context) (*corev1.ConfigMap, *configDocument, error) {
	cm, err := h.store.Get(c, configMapName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, defaultConfigDocument(), nil
		}
		return nil, nil, err
	}

	doc, err := decodeConfigDocument(cm)
	if err != nil {
		return nil, nil, err
	}
	return cm, doc, nil
}
