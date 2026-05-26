package common

import "strings"

const (
	DefaultDisasterSystemNamespace = "disaster-system"
	VeleroNamespace                = "velero"
	DisasterSystemLabel            = "testudo.softcdata.com/storage"
	DisasterSystemLabelValue       = "true"
)

var DisasterSystemNamespace = DefaultDisasterSystemNamespace

func SetDisasterSystemNamespace(namespace string) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = DefaultDisasterSystemNamespace
	}
	DisasterSystemNamespace = namespace
}
