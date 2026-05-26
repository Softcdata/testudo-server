package configs

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/spf13/viper"
)

const DefaultConfigPath = "configs/config.yaml"

var (
	Cfg *Config
)

type Config struct {
	Server  ServerConfig  `yaml:"server" mapstructure:"server"`
	Log     LogConfig     `yaml:"log" mapstructure:"log"`
	JWT     JWTConfig     `yaml:"jwt" mapstructure:"jwt"`
	Swagger SwaggerConfig `yaml:"swagger" mapstructure:"swagger"`
	License LicenseConfig `yaml:"license" mapstructure:"license"`
}

type ServerConfig struct {
	Host      string `yaml:"host" mapstructure:"host"`
	Port      int    `yaml:"port" mapstructure:"port"`
	Env       string `yaml:"env" mapstructure:"env"`
	Namespace string `yaml:"namespace" mapstructure:"namespace"`
}

type LogConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`
	Output string `yaml:"output" mapstructure:"output"`
}

type JWTConfig struct {
	Secret        string        `yaml:"secret" mapstructure:"secret"`
	Timeout       time.Duration `yaml:"timeout" mapstructure:"timeout"`
	Realm         string        `yaml:"realm" mapstructure:"realm"`
	AccessExpire  time.Duration `yaml:"access_expire" mapstructure:"access_expire"`
	RefreshExpire time.Duration `yaml:"refresh_expire" mapstructure:"refresh_expire"`
}

type SwaggerConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type LicenseConfig struct {
	Enabled   bool   `yaml:"enabled" mapstructure:"enabled"`
	Namespace string `yaml:"namespace" mapstructure:"namespace"`
	CAPath    string `yaml:"caPath" mapstructure:"caPath"`
}

func LoadConfig() error {
	v := viper.New()
	v.SetConfigFile(DefaultConfigPath)
	v.SetConfigType("yaml")
	v.SetDefault("server.namespace", common.DefaultDisasterSystemNamespace)
	v.SetDefault("swagger.enabled", false)
	v.SetDefault("license.enabled", true)
	v.SetDefault("license.namespace", "")
	v.SetDefault("license.caPath", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	err := v.ReadInConfig()
	if err != nil {
		return err
	}

	err = v.Unmarshal(&Cfg)
	if err != nil {
		return err
	}

	common.SetDisasterSystemNamespace(Cfg.Server.Namespace)
	Cfg.Server.Namespace = common.DisasterSystemNamespace
	if strings.TrimSpace(Cfg.License.Namespace) == "" {
		Cfg.License.Namespace = common.DisasterSystemNamespace
	}

	return nil
}

func Validate() error {
	if Cfg.Server.Host == "" {
		return fmt.Errorf("server host is required")
	}
	if Cfg.Server.Port == 0 {
		return fmt.Errorf("server port is required")
	}
	if Cfg.Server.Namespace == "" {
		return fmt.Errorf("server namespace is required")
	}
	return nil
}

func GetLogLevel() hlog.Level {
	level := Cfg.Log.Level
	switch level {
	case "error":
		return hlog.LevelError
	case "debug":
		return hlog.LevelDebug
	default:
		return hlog.LevelInfo
	}
}
