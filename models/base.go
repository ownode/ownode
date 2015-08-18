package models

import (
	"time"
)

type Base struct {
    CreatedAt time.Time `gorm:"created_at" json:"created_at"`
    UpdatedAt time.Time `gorm:"updated_at" json:"-"`
}

func (b *Base) BeforeUpdate() (err error) {
    b.UpdatedAt = time.Now().UTC()
    return
}

func (b *Base) BeforeCreate() (err error) {
	b.CreatedAt = time.Now().UTC()
    b.UpdatedAt = b.CreatedAt
    return
}