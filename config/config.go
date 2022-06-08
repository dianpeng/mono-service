package config

import (
	"os"
)

// Router configuration.
type KV struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type Router struct {
	Path   string   `yaml:"path"`
	Host   string   `yaml:"host"`
	Method []string `yaml:"method"`
	Query  []KV     `yaml:"query"`
}

type Service struct {
	Name                string   `yaml:"name"`
	Tag                 string   `yaml:"tag"`
	Router              *Router  `yaml:"router"`
	Method              []string `yaml:"method"`
	Policy              string   `yaml:"policy"`
	MaxSessionCacheSize int      `yaml:"max_session_cache_size"`

	// customized configuration for service itself
	Config map[string]interface{} `yaml:"config"`
}

type VHostResource struct {
	HttpClientPoolMaxSize  int64 `yaml:"http_client_pool_max_size"`
	HttpClientTimeout      int64 `yaml:"http_client_timeout"`
	HttpClientMaxDrainSize int64 `yaml:"http_client_max_drain_size"`
}

type VHost struct {
	Name          string        `yaml:"name"`
	Endpoint      string        `yaml:"endpoint"`
	ReadTimeout   int64         `yaml:"read_timeout"`
	WriteTimeout  int64         `yaml:"write_timeout"`
	MaxHeaderSize int           `yaml:"max_header_size"`
	LogFormat     string        `yaml:"log_format"`
	ErrorStatus   int           `yaml:"error_status"`
	ErrorBody     string        `yaml:"error_body"`
	RejectStatus  int           `yaml:"reject_status"`
	RejectBody    string        `yaml:"reject_body"`
	ServiceList   []*Service    `yaml:"service"`
	Resource      VHostResource `yaml:"resource"`
}

type Config struct {
	VHostList []*VHost `yaml:"vhost"`
}

func NotZeroInt(v int, d int) int {
	if v == 0 {
		return d
	}
	return v
}

func NotZeroInt64(v int64, d int64) int64 {
	if v == 0 {
		return d
	}
	return v
}

func NotZeroStr(v string, d string) string {
	if v == "" {
		return d
	}
	return v
}

func (s *Service) GetConfig(name string) interface{} {
	v, ok := s.Config[name]
	if !ok {
		return nil
	}
	return v
}

func (s *Service) GetConfigInt(name string) (int, bool) {
	v := s.GetConfig(name)
	if v == nil {
		return 0, false
	}
	vv, ok := v.(int)
	if !ok {
		return 0, false
	}
	return vv, true
}

func (s *Service) GetConfigIntDefault(name string, def int) int {
	v, ok := s.GetConfigInt(name)
	if !ok {
		return def
	}
	return v
}

func (s *Service) GetConfigInt64(name string) (int64, bool) {
	v := s.GetConfig(name)
	if v == nil {
		return 0, false
	}
	vv, ok := v.(int64)
	if !ok {
		return 0, false
	}
	return vv, true
}

func (s *Service) GetConfigInt64Default(name string, def int64) int64 {
	v, ok := s.GetConfigInt64(name)
	if !ok {
		return def
	}
	return v
}

func (s *Service) GetConfigString(name string) (string, bool) {
	v := s.GetConfig(name)
	if v == nil {
		return "", false
	}
	vv, ok := v.(string)
	if !ok {
		return "", false
	}
	return vv, true
}

func (s *Service) GetConfigStringDefault(name string, def string) string {
	v, ok := s.GetConfigString(name)
	if !ok {
		return def
	}
	return v
}

func (s *Service) GetConfigBool(name string) (bool, bool) {
	v := s.GetConfig(name)
	if v == nil {
		return false, false
	}
	vv, ok := v.(bool)
	if !ok {
		return false, false
	}
	return vv, true
}

func (s *Service) GetConfigBoolDefault(name string, def bool) bool {
	v, ok := s.GetConfigBool(name)
	if !ok {
		return def
	}
	return v
}

// Configuration of Hservice. The configuration of hservice is based on yaml
// syntax. Simply marshal the text into a structure
func ParseConfig(data string) (*Config, error) {
	r := &Config{}
	err := loadYaml(data, &r)
	return r, err
}

func ParseConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(string(data))
}
