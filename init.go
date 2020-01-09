package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

var (
	db *gorp.DbMap
)

func init() {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	// db, err := sql.Open("mysql", "tcp:localhost:3306*secu/mysql/123456")
	usr := conf.Args.Database.UserName
	pwd := conf.Args.Database.Password
	host := conf.Args.Database.Host
	port := conf.Args.Database.Port
	sch := conf.Args.Database.Schema
	mysql, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?readTimeout=12h&writeTimeout=12h", usr, pwd, host, port, sch))
	if err != nil {
		logrus.Panicln("sql.Open failed", err)
	}

	mysql.SetMaxOpenConns(16)
	mysql.SetMaxIdleConns(5)
	mysql.SetConnMaxLifetime(time.Second * 15)

	// construct a gorp DbMap
	db = &gorp.DbMap{Db: mysql, Dialect: gorp.MySQLDialect{"InnoDB", "utf8"}}

	err = mysql.Ping()
	if err != nil {
		logrus.Panic("Failed to ping db", err)
	}

	logFormatter := new(logrus.TextFormatter)
	logFormatter.FullTimestamp = true
	logFormatter.TimestampFormat = "2006-01-02 15:04:05"
	logrus.SetFormatter(logFormatter)
}
