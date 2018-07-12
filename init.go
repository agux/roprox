package main

import (
	"database/sql"
	"time"

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
	mysql, err := sql.Open("mysql", "mysql:123456@/secu")
	if err != nil {
		logrus.Panicln("sql.Open failed", err)
	}

	mysql.SetMaxOpenConns(64)
	mysql.SetMaxIdleConns(64)
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
