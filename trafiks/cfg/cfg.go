package cfg

import (
	"flag"
	"log"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewConf),
)

type Config struct {
	LogLevel    string         `mapstructure:"log_level"`
	LogOutput   string         `mapstructure:"log_output"`
	ServerPort  string         `mapstructure:"server_port"`
	TLSPort     string         `mapstructure:"tls_port"`
	AppBaseURL  string         `mapstructure:"app_base_url"`
	Environment string         `mapstructure:"environment"`
	Database    database       `mapstructure:"database"`
	JwtSecret   string         `mapstructure:"jwt_secret"`
	Redis       RedisConf      `mapstructure:"redis"`
	Dashboard   DashboardCfg   `mapstructure:"dashboard"`
	Bootstrap   BootstrapCfg   `mapstructure:"bootstrap"`
	Docker      DockerConf     `mapstructure:"docker"`
	Kubernetes  KubernetesConf `mapstructure:"kubernetes"`
	Proxy       ProxyConf      `mapstructure:"proxy"`
}

type DashboardCfg struct {
	Enabled *bool `mapstructure:"enabled"`
}

type BootstrapCfg struct {
	User BootstrapUser `mapstructure:"user"`
}

type BootstrapUser struct {
	Email     string `mapstructure:"email"`
	Password  string `mapstructure:"password"`
	FirstName string `mapstructure:"first_name"`
	LastName  string `mapstructure:"last_name"`
}

type RedisConf struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Db       int    `json:"db"`
}

type DockerConf struct {
	SocketPath string `mapstructure:"socket_path"`
}

type KubernetesConf struct {
	KubeconfigPath string `mapstructure:"kubeconfig_path"` // Empty = in-cluster config
}

type ProxyConf struct {
	Server    ProxyServerConf    `mapstructure:"server"`
	Transport ProxyTransportConf `mapstructure:"transport"`
}

type ProxyServerConf struct {
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
}

type ProxyTransportConf struct {
	MaxIdleConns        int           `mapstructure:"max_idle_connections"`
	MaxIdleConnsPerHost int           `mapstructure:"max_idle_connections_per_host"`
	IdleConnTimeout     time.Duration `mapstructure:"idle_connection_timeout"`
	DialTimeout         time.Duration `mapstructure:"dial_timeout"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout"`
}

type database struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	Name       string `mapstructure:"name"`
	SSLMode    string `mapstructure:"ssl_mode"`
	LogEnabled *bool  `mapstructure:"log_enabled"`
}

var cfg *Config

func NewConf() (*Config, error) {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.Parse()

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath("$HOME/.config/trafiks")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found; using default values")
		} else {
			return nil, err
		}
	}

	var conf Config
	if err := viper.Unmarshal(&conf); err != nil {
		return nil, err
	}

	setDefaults(&conf)

	cfg = &conf

	return &conf, nil
}

func setDefaults(conf *Config) {
	if conf.LogLevel == "" {
		conf.LogLevel = "debug"
	}
	if conf.LogOutput == "" {
		conf.LogOutput = "stdout"
	}
	if conf.ServerPort == "" {
		conf.ServerPort = "8890"
	}
	if conf.TLSPort == "" {
		conf.TLSPort = "8444"
	}
	if conf.AppBaseURL == "" {
		conf.AppBaseURL = "localproxy.test"
	}
	if conf.Environment == "" {
		conf.Environment = "local"
	}

	// Database defaults
	if conf.Database.Host == "" {
		conf.Database.Host = "localhost"
	}
	if conf.Database.Port == 0 {
		conf.Database.Port = 5436
	}
	if conf.Database.User == "" {
		conf.Database.User = "admin"
	}
	if conf.Database.Password == "" {
		conf.Database.Password = "password"
	}
	if conf.Database.Name == "" {
		conf.Database.Name = "trafiks"
	}
	if conf.Database.SSLMode == "" {
		conf.Database.SSLMode = "disable"
	}
	if conf.Database.LogEnabled == nil {
		enabled := true
		conf.Database.LogEnabled = &enabled
	}

	// Redis defaults
	if conf.Redis.Host == "" {
		conf.Redis.Host = "localhost:6381"
	}
	if conf.Redis.Db == 0 && conf.Redis.Host != "" {
		conf.Redis.Db = 0
	}

	// JWT Secret default (generate a simple one if not set)
	if conf.JwtSecret == "" {
		conf.JwtSecret = "2pPLTGtIGEDhvIDPJBYp/c8gzV5bRacq9QE7pDpBEdM26AmONv0+6crsf+suz/Nn"
	}

	if conf.Dashboard.Enabled == nil {
		enabled := true
		conf.Dashboard.Enabled = &enabled
	}

	// Bootstrap user defaults
	if conf.Bootstrap.User.Email == "" {
		conf.Bootstrap.User.Email = "admin@trafiks.local"
	}
	if conf.Bootstrap.User.Password == "" {
		conf.Bootstrap.User.Password = "admin123"
	}
	if conf.Bootstrap.User.FirstName == "" {
		conf.Bootstrap.User.FirstName = "admin"
	}
	if conf.Bootstrap.User.LastName == "" {
		conf.Bootstrap.User.LastName = "user"
	}

	if conf.Docker.SocketPath == "" {
		// conf.Docker.SocketPath = "unix:///Users/olalekanodukoya/.colima/k8s/docker.sock"
		conf.Docker.SocketPath = "unix:///var/run/docker.sock"
	}

	// Proxy server defaults
	if conf.Proxy.Server.ShutdownTimeout == 0 {
		conf.Proxy.Server.ShutdownTimeout = 10 * time.Second
	}
	if conf.Proxy.Server.ReadTimeout == 0 {
		conf.Proxy.Server.ReadTimeout = 10 * time.Second
	}
	if conf.Proxy.Server.WriteTimeout == 0 {
		conf.Proxy.Server.WriteTimeout = 10 * time.Second
	}
	if conf.Proxy.Server.IdleTimeout == 0 {
		conf.Proxy.Server.IdleTimeout = 120 * time.Second
	}

	// Proxy transport defaults
	if conf.Proxy.Transport.MaxIdleConns == 0 {
		conf.Proxy.Transport.MaxIdleConns = 100
	}
	if conf.Proxy.Transport.MaxIdleConnsPerHost == 0 {
		conf.Proxy.Transport.MaxIdleConnsPerHost = 20
	}
	if conf.Proxy.Transport.IdleConnTimeout == 0 {
		conf.Proxy.Transport.IdleConnTimeout = 90 * time.Second
	}
	if conf.Proxy.Transport.DialTimeout == 0 {
		conf.Proxy.Transport.DialTimeout = 5 * time.Second
	}
	if conf.Proxy.Transport.RequestTimeout == 0 {
		conf.Proxy.Transport.RequestTimeout = 30 * time.Second
	}
}

func GetConf() *Config {
	if cfg == nil {
		cfg = &Config{}
		setDefaults(cfg)
	}

	return cfg
}
