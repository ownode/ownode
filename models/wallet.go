package models

import (
	"github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
    "database/sql"
    // "github.com/ownode/services"
)

type Wallet struct {
	ID  uint `gorm:"primary_key" json:"-"`
	ObjectID string `gorm:"object_id" json:"id" sql:"not null;unique"`
	Identity Identity `json:"identity"`
	IdentityID  sql.NullInt64 `json:"-"`
	Handle string  	`json:"handle"`
    Password string	 `json:"-"`
	Objects []Object `json:"-"`
	Lock bool `json:"lock"`
	Base
}

// create wallet
func CreateWallet(db *gorm.DB, wallet *Wallet) error {
	return db.Create(wallet).Error
}

// find wallet by handle
func FindWalletByHandle(db *gorm.DB, handle string) (Wallet, bool, error) {
	result := Wallet{}
	err := db.Where(&Wallet{ Handle: handle }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		} 
		return result, false, err
	}
	return result, true, nil
}

// find wallet by object id
func FindWalletByObjectID(db *gorm.DB, objectID string) (Wallet, bool, error) {
	result := Wallet{}
	err := db.Preload("Identity").Where(&Wallet{ ObjectID: objectID }).First(&result).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			return result, false, nil
		} 
		return result, false, err
	}
	
	return result, true, nil
}
