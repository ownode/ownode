package services

import (
    "gopkg.in/mgo.v2"
    "errors"
    "github.com/jinzhu/gorm"
    _ "github.com/lib/pq"
)

var (
	MongoDBName string
	IdentityColName string
	ServiceColName string
	WalletColName string
	TokenColName string
)

func init () {
	MongoDBName = GetEnvOrDefault("OWNODE_DB_NAME", "ownode")
	IdentityColName = GetEnvOrDefault("OWNODE_IDENTITY_COL_NAME", "Identities")
	ServiceColName = GetEnvOrDefault("OWNODE_SERVICE_COL_NAME", "services")
	WalletColName = GetEnvOrDefault("OWNODE_WALLET_COL_NAME", "wallets")
	TokenColName = GetEnvOrDefault("OWNODE_TOKEN_COL_NAME", "tokens")
}

type DB struct {
	session *mgo.Session
	pgDB *gorm.DB
}

// connects to mongo db and return session
func (db *DB) ConnectToMongo() (*mgo.Session, error) {
	session, err := mgo.Dial("localhost")		// TODO: get from env variable
    if err != nil {
        return &mgo.Session{}, errors.New("unable to connect to database")
    }
    db.session = session
    return session, nil
}

// returns mongo session
func (db *DB) GetMongoSession() *mgo.Session {
	return db.session
}

// close mongo database session
func (db *DB) CloseMongoSession() {
	db.session.Close()
}

// connect to postgres db
func (db *DB) ConnectToPostgres(args string) (*gorm.DB, error) {
	
	dbObj, err := gorm.Open("postgres", args)
	if err != nil {
		return &dbObj, errors.New("unable to connect to postgres. reason: " + err.Error())
	}
	
	if err = dbObj.DB().Ping(); err != nil {
		return &dbObj, errors.New("unable to ping postgres. reason: " + err.Error())
	}

	db.pgDB = &dbObj
	// db.pgDB.LogMode(true)
	return db.pgDB, err
}

func (dbM *DB) setDB(db *gorm.DB) {
	_ = db.Exec("set time zone 'utc';")
}

// get postgres db handle
func (db *DB) GetPostgresHandle() *gorm.DB {
	db.setDB(db.pgDB)
	return db.pgDB
}

// get postgres transaction with isolation set to repeatable read
func (db *DB) GetPostgresHandleWithRepeatableReadTrans() (*gorm.DB, error) {
	tx := db.GetPostgresHandle().Begin()
	err := tx.Exec(`set transaction isolation level repeatable read`).Error
	if err != nil {
		tx.Rollback()
	    return tx, err
	}
	return tx, nil
}

// close postgres database
func (db *DB) ClosePostgresHandle() {
	db.pgDB.Close()
}