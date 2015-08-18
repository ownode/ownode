package models

import (
	"time"
	"github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
)

type Token struct {
	ID  uint `gorm:"primary_key" json:"-"`
	Service Service `gorm:"client_id" json:"service"`
	Token string `json:"token" sql:"not null;unique"` 
	Type string `json:"type"`
	Auth *Authorization `json:"authorization,omitempty"`
	ExpiresIn time.Time 	`json:"expires_in"`
	CreatedAt time.Time 	`json:"created_at"`
    UpdatedAt time.Time 	`json:"-"`
}

// create a token
func CreateToken(db *gorm.DB, token *Token) error {
	return db.Create(token).Error
}

// find a token
func FindToken(db *gorm.DB, token string) error {
	return nil
}
