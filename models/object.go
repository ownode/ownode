package models

import (
	// "time"
	"github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
    "database/sql"
    // "github.com/ownode/services"
)

var (
	ObjectValue = "obj_value"
	ObjectValueless = "obj_valueless"
	ObjectOpenDefault = "open"
	ObjectOpenTimed = "open_timed"
	ObjectOpenPin = "open_pin"
)

type Object struct {
	ID  uint `gorm:"primary_key" json:"-" sql:"type:bigserial"`
	ObjectID string `gorm:"object_id" json:"id" sql:"not null;unique"`
	Pin string  `gorm:"pin" json:"pin" sql:"not null;unique"`
	Type string `gorm:"type" json:"type"`
	Wallet Wallet `json:"wallet"`
	WalletID  sql.NullInt64 `json:"-"`
	Service Service `json:"service"`
	ServiceID  sql.NullInt64 `json:"-"`
	Balance float64  `gorm:"balance" json:"balance"`
	Meta string `gorm:"meta" json:"meta" sql:"type:text"`
	Open bool `gorm:"open" json:"open"`
	OpenMethod string `gorm:"open_method" json:"open_method,omitempty"`
	OpenTime int64 `gorm:"open_time" json:"open_time,omitempty"`
	OpenPin string `gorm:"open_pin" json:"-"`
	Base
}

// create an object
func CreateObject(db *gorm.DB, object *Object) error {
	return db.Create(object).Error
}

// find object by object id
func FindObjectByObjectID(db *gorm.DB, objectID string) (Object, bool, error) {
	result := Object{}
	err := db.Preload("Service.Identity").Preload("Wallet.Identity").Where(&Object{ ObjectID: objectID }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		} 
		return result, false, err
	}
	return result, true, nil
}

// find object by object id or by pin
func FindObjectByObjectIDOrPin(db *gorm.DB, IDOrPin string) (Object, bool, error) {
	result := Object{}
	err := db.Preload("Service.Identity").Preload("Wallet.Identity").Or(&Object{ ObjectID: IDOrPin }).Where(&Object{ Pin: IDOrPin }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		} 
		return result, false, err
	}
	return result, true, nil
}

// find all objects contained in a list of object ids
func FindAllObjectsByObjectID(db *gorm.DB, objects []string) ([]Object, error) {
	result := []Object{}
	return result, db.Preload("Service.Identity").Preload("Wallet.Identity").Where("object_id IN (?)", objects).Find(&result).Error
}
