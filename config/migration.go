package config

import (
	"github.com/ownode/models"
	"github.com/ownode/services"
)

func PostgresAutoMigration(db *services.DB) {
	db.GetPostgresHandle().AutoMigrate(&models.Token{}, &models.Service{}, &models.Identity{}, &models.Wallet{}, &models.Object{})
	services.Println("Migration complete!")
}