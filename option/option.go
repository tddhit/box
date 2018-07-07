package option

type Api struct {
	Path   string              `yaml:"path"`
	Method string              `yaml:"method"`
	Header map[string][]string `yaml:"header"`
	Limit  float64             `yaml:"limit"`
	Burst  int                 `yaml:"burst"`
}

type Client struct {
	HTTPVersion     string `yaml:"httpVersion"`
	ConnectTimeout  int64  `yaml:"connectTimeout"`
	ReadTimeout     int64  `yaml:"readTimeout"`
	IdleConnTimeout int64  `yaml:"idleConnTimeout"`
	KeepAlive       int64  `yaml:"keepAlive"`
	MaxIdleConns    int    `yaml:"maxIdleConns"`
}

type CircuitBreaker struct {
	MaxRequests   uint32  `yaml:"maxRequests"`
	Interval      uint32  `yaml:"interval"`
	Timeout       uint32  `yaml:"timeout"`
	TotalRequests uint32  `yaml:"totalRequests"`
	FailureRatio  float64 `yaml:"failureRatio"`
}

type Location struct {
	Method  string
	Pattern string
}

type Upstream struct {
	Enable         bool           `yaml:"enable"`
	IsProxy        bool           `yaml:"isProxy"`
	Locations      []Location     `yaml:"locations"`
	Method         string         `yaml:"method"`
	Registry       string         `yaml:"registry"`
	Api            map[string]Api `yaml:"api"`
	Client         Client         `yaml:"client"`
	CircuitBreaker CircuitBreaker `yaml:"circuitBreaker"`
}

type Server struct {
	HTTPVersion      string         `yaml:"httpVersion"`
	Registry         string         `yaml:"registry"`
	MasterAddr       string         `yaml:"masterAddr"`
	WorkerAddr       string         `yaml:"workerAddr"`
	TransportAddr    string         `yaml:"addr"`
	TracingAgentAddr string         `yaml:"tracingAgentAddr"`
	PIDPath          string         `yaml:"pidPath"`
	WorkerNum        int            `yaml:"workerNum"`
	ReadTimeout      int64          `yaml:"readTimeout"`
	WriteTimeout     int64          `yaml:"writeTimeout"`
	IdleTimeout      int64          `yaml:"idleTimeout"`
	Api              map[string]Api `yaml:"api"`
}
