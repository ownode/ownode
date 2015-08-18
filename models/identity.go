package models

import (
	"github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
    // "database/sql"
    // "github.com/ownode/services"
)

type Identity struct {
	ID	uint `gorm:"primary_key" json:"-"`
	ObjectID string `gorm:"object_id" json:"id" sql:"not null;unique"`
	FullName string `json:"full_name"`
	Email string `json:"-"`
	
	// issuer specific field
	Issuer bool `json:"issuer,omitempty"`
	SoulBalance float64 `json:"-"`
	ObjectName  string `json:"object_name,omitempty"`
	BaseCurrency string	`bson:"base_currency" json:"base_currency,omitempty"`
	
	Base
}

// create an identity
func CreateIdentity(db *gorm.DB, identity *Identity) error {
	return db.Create(identity).Error
}

// find an identity by email
func FindIdentityByEmail(db *gorm.DB, email string) (Identity, bool, error) {
	result := Identity{}
	err := db.Where(&Identity{ Email: email }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		}
		return result, false, err
	}
	return result, true, nil
}


// find an identity by id
func FindIdentityById(db *gorm.DB, id uint) (Identity, bool, error) {
	result := Identity{}
	err := db.Where(&Identity{ ID: id }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		}
		return result, false, err
	}
	return result, true, nil
}

// find an identity by object id
func FindIdentityByObjectID(db *gorm.DB, id string) (Identity, bool, error) {
	result := Identity{}
	err := db.Where(&Identity{ ObjectID: id }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		}
		return result, false, err
	}

	return result, true, nil
}

// add to a identities soul amount
func AddToSoulByObjectID (db *gorm.DB, id string, incrVal float64) (Identity, error) {
	
	identity := Identity{}
	tx := db.Begin()

	err := tx.Exec(`set transaction isolation level repeatable read`).Error
	if err != nil {
		tx.Rollback()
	    return identity, err
	}

	// get identity
	if err = tx.Where(&Identity{ ObjectID: id }).First(&identity).Error; err != nil {
		tx.Rollback()
		return identity, err
	}

	// add to identities soul amount
	identity.SoulBalance = identity.SoulBalance + incrVal

	// update identity
	tx.Save(&identity)

	tx.Commit()
	return identity, nil
}

// find by object name
func FindIdentityByObjectName(db *gorm.DB, name string) (Identity, bool, error) {
	result := Identity{}
	err := db.Where(&Identity{ ObjectName: name }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		}
		return result, false, err
	}
	return result, true, nil
}

