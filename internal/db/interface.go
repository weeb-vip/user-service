package db

import "gorm.io/gorm"

type DB interface {
	GetDB() *gorm.DB
}
