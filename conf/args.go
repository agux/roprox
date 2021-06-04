package conf

import (
	"go/build"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Args Global Application Arguments
var Args Arguments

var vp *viper.Viper

//Arguments arguments struct type
type Arguments struct {
	LogLevel               string  `mapstructure:"log_level"`
	ScannerPoolSize        int     `mapstructure:"scanner_pool_size"`
	ScannerMaxRetry        int     `mapstructure:"scanner_max_retry"`
	LocalProbeSize         int     `mapstructure:"local_probe_size"`
	LocalProbeInterval     int     `mapstructure:"local_probe_interval"`
	LocalProbeTimeout      int     `mapstructure:"local_probe_timeout"`
	LocalProbeRetry        int     `mapstructure:"local_probe_retry"`
	GlobalProbeSize        int     `mapstructure:"global_probe_size"`
	GlobalProbeInterval    int     `mapstructure:"global_probe_interval"`
	GlobalProbeTimeout     int     `mapstructure:"global_probe_timeout"`
	GlobalProbeRetry       int     `mapstructure:"global_probe_retry"`
	EvictionTimeout        int     `mapstructure:"eviction_timeout"`
	EvictionInterval       int     `mapstructure:"eviction_interval"`
	EvictionScoreThreshold float32 `mapstructure:"eviction_score_threshold"`

	Logging struct {
		LogFilePath string `mapstructure:"log_file_path"`
	}

	Network struct {
		MasterProxyAddr                 string  `mapstructure:"master_proxy_addr"`
		DefaultUserAgent                string  `mapstructure:"default_user_agent"`
		HTTPTimeout                     int     `mapstructure:"http_timeout"`
		HTTPRetry                       int     `mapstructure:"http_retry"`
		RotateProxyScoreThreshold       float64 `mapstructure:"rotate_proxy_score_threshold"`
		RotateProxyGlobalScoreThreshold float64 `mapstructure:"rotate_proxy_global_score_threshold"`
	}

	WebDriver struct {
		Timeout       int    `mapstructure:"timeout"`
		Headless      bool   `mapstructure:"headless"`
		NoImage       bool   `mapstructure:"no_image"`
		MaxRetry      int    `mapstructure:"max_retry"`
		WorkingFolder string `mapstructure:"working_folder"`
	}

	DataSource struct {
		UserAgents        string `mapstructure:"user_agents"`
		UserAgentLifespan int    `mapstructure:"user_agent_lifespan"`

		SpysOne struct {
			ProxyMode       string `mapstructure:"proxy_mode"`
			Headless        bool   `mapstructure:"headless"`
			RefreshInterval int    `mapstructure:"refresh_interval"`
			Retry           int    `mapstructure:"retry"`
		}
		HideMyName struct {
			ProxyMode       string `mapstructure:"proxy_mode"`
			Headless        bool   `mapstructure:"headless"`
			RefreshInterval int    `mapstructure:"refresh_interval"`
			Retry           int    `mapstructure:"retry"`
			HomePageTimeout int    `mapstructure:"homepage_timeout"`
		}
	}

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Schema   string `mapstructure:"schema"`
		UserName string `mapstructure:"user_name"`
		Password string `mapstructure:"password"`
	}
}

func init() {
	vp = viper.New()
	setDefaults()
	vp.SetConfigName("roprox") // name of config file (without extension)
	gopath := os.Getenv("GOPATH")
	if "" == gopath {
		gopath = build.Default.GOPATH
	}
	vp.AddConfigPath(filepath.Join(gopath, "bin"))
	vp.AddConfigPath(".") // optionally look for config in the working directory
	// vp.AddConfigPath("$HOME")
	err := vp.ReadInConfig()
	if err != nil {
		log.Panicf("config file error: %+v", err)
	}
	err = vp.Unmarshal(&Args)
	if err != nil {
		log.Panicf("config file error: %+v", err)
	}
	// log.Printf("Configuration: %+v", Args)
	//vp.WatchConfig()
	//vp.OnConfigChange(func(e fsnotify.Event) {
	//	fmt.Println("Config file changed:", e.Name)
	//})
	checkConfig()
}

func checkConfig() {
	//check if config parameters are valid
}

func setDefaults() {
	vp.SetDefault("log_level", "info")
	vp.SetDefault("DataSource.SpysOne.proxy_mode", "master")
	vp.SetDefault("DataSource.SpysOne.headless", true)
	vp.SetDefault("DataSource.SpysOne.refresh_interval", 60)
	vp.SetDefault("DataSource.HideMyName.proxy_mode", "master")
	vp.SetDefault("DataSource.HideMyName.headless", false)
	vp.SetDefault("DataSource.HideMyName.refresh_interval", 60)

	// Args.LogLevel = "info"
}

// ConfigFileUsed returns the file used to populate the config registry.
func ConfigFileUsed() string {
	return vp.ConfigFileUsed()
}
