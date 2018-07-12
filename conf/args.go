package conf

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Args Global Application Arguments
var Args Arguments

//Arguments arguments struct type
type Arguments struct {
	LogLevel            string `mapstructure:"log_level"`
	MasterProxyAddr     string `mapstructure:"master_proxy_addr"`
	HTTPRetry           int    `mapstructure:"http_retry"`
	ScannerPoolSize     int    `mapstructure:"scanner_pool_size"`
	ScannerMaxRetry     int    `mapstructure:"scanner_max_retry"`
	CheckerPoolSize     int    `mapstructure:"checker_pool_size"`
	CheckInterval   int    `mapstructure:"check_interval"`
	CheckTimeout int    `mapstructure:"check_timeout"`
	//TODO logrus log to file
}

func init() {
	setDefaults()
	viper.SetConfigName("roprox") // name of config file (without extension)
	viper.AddConfigPath("$GOPATH/bin")
	viper.AddConfigPath(".") // optionally look for config in the working directory
	viper.AddConfigPath("$HOME")
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Errorf("config file error: %+v", err)
		return
	}
	err = viper.Unmarshal(&Args)
	if err != nil {
		logrus.Errorf("config file error: %+v", err)
		return
	}
	logrus.Printf("Configuration: %+v", Args)
	switch Args.LogLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warning":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	}
	//viper.WatchConfig()
	//viper.OnConfigChange(func(e fsnotify.Event) {
	//	fmt.Println("Config file changed:", e.Name)
	//})
	checkConfig()
}

func checkConfig() {
	//check if config parameters are valid
}

func setDefaults() {
	Args.LogLevel = "info"
}
