package data

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/logging"
	"github.com/go-gorp/gorp"

	//the mysql driver
	_ "github.com/go-sql-driver/mysql"
)

var (
	//DB the database instance
	DB  *gorp.DbMap
	log = logging.Logger
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
		log.Panicln("sql.Open failed", err)
	}

	mysql.SetMaxOpenConns(16)
	mysql.SetMaxIdleConns(5)
	mysql.SetConnMaxLifetime(time.Second * 15)

	// construct a gorp DbMap
	DB = &gorp.DbMap{Db: mysql, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8"}}

	err = mysql.Ping()
	if err != nil {
		log.Panic("Failed to ping db", err)
	}
}
