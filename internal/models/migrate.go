package models

import (
	"log"

	"gorm.io/gorm"
)

// AutoMigrate handles the database schema migration
func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		&User{},
		&Profile{},
		&Property{},
		&PropertyImage{},
		&LeaseRequest{},
		&LeaseAgreement{},
		&Invoice{},
		&MaintenanceRequest{},
		&MaintenanceUpdate{},
		&Payment{},
		&Notification{},
		&Invitation{},
		&Rating{},
		&Message{},
		&Wallet{},
		&WalletTransaction{},
		&SaleReservation{},
		&TransactionDocument{},
		&Dispute{},
		&AuditEvent{},
		&PropertyTransaction{},
		&PaymentMilestone{},
		&SavedProperty{},
		&PropertyOwnership{},
	)

	if err != nil {
		return err
	}

	log.Println("Database migration completed successfully.")
	return nil
}
