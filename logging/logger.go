package logging

import (
	"io"
	"os"

	"github.com/carusyte/roprox/conf"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

//Logger the global logger for this project
var Logger = logrus.New()

const (
	//DateFormat is project-standard date format.
	DateFormat     = "2006-01-02"
	//TimeFormat is project-standard time format.
	TimeFormat     = "15:04:05"
	//DateTimeFormat is project-standard datetime format.
	DateTimeFormat = "2006-01-02 15:04:05"
)

func init() {
	switch conf.Args.LogLevel {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warning":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		Logger.SetLevel(logrus.FatalLevel)
	case "panic":
		Logger.SetLevel(logrus.PanicLevel)
	}

	Logger.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: DateTimeFormat,
		FullTimestamp:   true,
		ForceFormatting: true,
		// ForceColors:     true,
	})
	if _, e := os.Stat(conf.Args.Logging.LogFilePath); e == nil {
		os.Remove(conf.Args.Logging.LogFilePath)
	}
	logFile, e := os.OpenFile(conf.Args.Logging.LogFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if e != nil {
		Logger.Panicln("failed to open log file", e)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	Logger.SetOutput(mw)
	logrus.RegisterExitHandler(func() {
		if logFile != nil {
			logFile.Close()
		}
	})
}
