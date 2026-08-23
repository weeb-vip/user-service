package db

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/weeb-vip/user-service/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	// Matched deliberately. Go retains only maxIdleConns; anything opened above
	// it is closed again as soon as the query finishes. At 100 open and 10 idle
	// this pool was not reusing 100 connections -- it held 10 and rebuilt up to
	// 90, paying a TCP connect, TLS handshake and Postgres auth per query
	// against RDS over the internet.
	//
	// 5 because the database is small: db.t4g.micro allows 79 connections in
	// total and roughly 36 pods share them. One pod configured for 100 could
	// exhaust the whole database on its own. A warm pool of 5 serves more
	// traffic than a churning pool of 100 while claiming a twentieth of it.
	maxIdleConns = 5
	maxOpenConns = 5
)

type SafeDBService struct {
	mu sync.Mutex
	db DB //nolint
}

type Service struct {
	db *gorm.DB
}

var dbservice = SafeDBService{ // nolint
	db: nil,
}

func (service *Service) setupSQLDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to connect database")
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(maxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(maxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// Bounded so a failover or DNS change is picked up without a restart, but long
	// enough that connections survive quiet periods and get reused.
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
}

func (service *Service) connect(cfg config.DBConfig) *gorm.DB {
	log.Println("Connecting to database...", cfg.Host, cfg.Port, cfg.DB)
	db, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DB, cfg.SSL)), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	service.setupSQLDB(db)

	service.db = db

	return db
}

func NewDBService() DB { //nolint
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	temp := &Service{}
	temp.connect(cfg.DBConfig)

	dbservice.SetDB(temp)

	return dbservice.GetDB()
}

func (service *Service) GetDB() *gorm.DB {
	return service.db
}

func (service *SafeDBService) GetDB() DB {
	service.mu.Lock()
	defer service.mu.Unlock()

	return service.db
}

func (service *SafeDBService) SetDB(db DB) {
	service.mu.Lock()
	defer service.mu.Unlock()

	service.db = db
}

func GetDBService() DB {
	if dbservice.GetDB() != nil {
		return dbservice.GetDB()
	}

	return NewDBService()
}
