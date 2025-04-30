package db

import (
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/weeb-vip/user/internal/snake_case"
	"github.com/weeb-vip/user/internal/ulid"
)

type BaseModel struct {
	ID        string    `gorm:"primary_key" column:"id"`
	CreatedAt time.Time `column:"created_at"`
	UpdatedAt time.Time `column:"updated_at"`
}

func (baseModel *BaseModel) BeforeCreate(db *gorm.DB) error {
	if baseModel.ID != "" {
		return nil
	}

	model := reflect.TypeOf(db.Statement.Model).String()

	modelName := model[strings.LastIndex(model, ".")+1:]
	modelName = snake_case.ToSnakeCase(modelName)

	baseModel.ID = ulid.New(modelName)

	return nil
}
