package data

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	//FIXME to be obsolete
	"gopkg.in/gorp.v2"
	"gorm.io/gorm"

	//the sqlite embed db driver
	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/types"
	_ "github.com/glebarez/go-sqlite"
	"gorm.io/driver/sqlite"

	//import mysql driver
	_ "github.com/go-sql-driver/mysql"
)

var (
	//DB the database instance
	DB     *gorp.DbMap // TO BE DEPRECATED
	GormDB *gorm.DB
	log    = logging.Logger
)

func init() {
	// improve the following switch to match case-insensitive
	switch strings.ToLower(conf.Args.Database.DbType) {
	case "mysql":
		initMySQL()
	case "sqlite":
		initSQLite()
	default:
		log.Panic("unsupported database type. please check 'conf.Args.Database.Type' in configuration file: ",
			conf.Args.Database.DbType)
	}
}

func initMySQL() {
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

func initSQLite() {
	var err error

	// Initialize GORM with the SQLite database.
	GormDB, err = gorm.Open(sqlite.Open(conf.Args.Database.Path), &gorm.Config{})
	if err != nil {
		log.Panicln("gorm.Open() failed", err)
	}

	if err = GormDB.AutoMigrate(
		&types.ProxyServer{},
		&types.UserAgent{},
		&types.NetworkTraffic{},
	); err != nil {
		log.Panicln("GORM auto migrate failure", err)
	}

	// Execute PRAGMA statements.
	if err = GormDB.Exec("PRAGMA synchronous = OFF").Error; err != nil {
		log.Panicln("failed to execute 'PRAGMA synchronous = OFF' ", err)
	}

	if err = GormDB.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		log.Panicln("failed to execute 'PRAGMA journal_mode = WAL' ", err)
	}
}
