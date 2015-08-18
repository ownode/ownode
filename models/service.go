package models

import (
	"github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
    "database/sql"
)

type Service struct {
	ID  uint `gorm:"primary_key" json:"-"`
	Name string `json:"name"`
	Identity *Identity `json:"identity"`
	IdentityID  sql.NullInt64 `json:"-"`
	ObjectID string `gorm:"object_id" json:"id" sql:"not null;unique"`
	Description string `json:"description"`
	ClientID string	`bson:"client_id" json:"-" sql:"not null;unique"`
	ClientSecret string	`bson:"client_secret" json:"-"`
    Base
}

// create a service
func CreateService(db *gorm.DB, service *Service) error {
	return db.Create(service).Error
}

// find a service by it's client id
func FindServiceByClientId(db *gorm.DB, clientId string) (Service, bool, error) {
	result := Service{}
	err := db.Preload("Identity").Where(&Service{ ClientID: clientId }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		} else {
			return result, false, err
		}
	}
	return result, true, nil
}

// find a service by id
func FindServiceByObjectID(db *gorm.DB, id string) (Service, bool, error) {
	result := Service{}
	err := db.Preload("Identity").Where(&Service{ ObjectID: id }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		} 
		return result, false, err
	}
	return result, true, nil
}
