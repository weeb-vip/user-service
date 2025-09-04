package db

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/weeb-vip/user-service/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	maxIdleConns = 10
	maxOpenConns = 100
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
	sqlDB.SetConnMaxLifetime(time.Hour)
}

func (service *Service) connect(cfg config.DBConfig) *gorm.DB {
	log.Println("Connecting to database...", cfg.Host, cfg.Port, cfg.DB)
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s&interpolateParams=true", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DB, cfg.SSL)), &gorm.Config{})

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
