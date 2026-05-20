package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"real-estate-backend/internal/models"
	"real-estate-backend/pkg/mail"
	"real-estate-backend/pkg/payment"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletService interface {
	GetWalletByUserID(userID uuid.UUID) (*models.Wallet, error)
	GetOrCreateWallet(userID uuid.UUID) (*models.Wallet, error)
	GetWalletTransactions(walletID uuid.UUID) ([]models.WalletTransaction, error)
	GetSmartSavingTimeline(userID uuid.UUID, propertyID uuid.UUID) (map[string]interface{}, error)
	SetPIN(userID uuid.UUID, pin string) (*models.Wallet, error)
	ChangePIN(userID uuid.UUID, currentPIN string, newPIN string) (*models.Wallet, error)
	Withdraw(userID uuid.UUID, amount float64, pin string) error
	PayReservationDeposit(userID uuid.UUID, propertyID uuid.UUID, pin string) error
	PayRent(userID uuid.UUID, propertyID uuid.UUID, pin string) error
	ActivateVirtualAccount(userID uuid.UUID) (*models.Wallet, error)
	InitializePayment(userID uuid.UUID, amount float64, callbackURL string, pin string) (*payment.InitializeResponse, error)
	ConfirmPayment(userID uuid.UUID, reference string) (*models.Wallet, error)
	PayInvoice(userID uuid.UUID, invoiceID uuid.UUID, pin string) error
	DemoFund(userID uuid.UUID, amount float64) error
}

type walletService struct {
	db              *gorm.DB
	paystackService payment.PaystackService
	mailService     mail.MailService
}

func NewWalletService(db *gorm.DB, paystackService payment.PaystackService, mailService mail.MailService) WalletService {
	return &walletService{db: db, paystackService: paystackService, mailService: mailService}
}

var walletPINPattern = regexp.MustCompile(`^\d{4}$`)

func (s *walletService) GetWalletByUserID(userID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (s *walletService) GetOrCreateWallet(userID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	var user models.User

	// Fetch user with profile
	if err := s.db.Preload("Profile").First(&user, userID).Error; err != nil {
		return nil, err
	}

	err := s.db.Where("user_id = ?", userID).First(&wallet).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 1. Create Paystack Customer first
		custRes, err := s.paystackService.CreateCustomer(user.Email, user.Profile.FullName, "", user.Profile.PhoneNumber)
		if err != nil || !custRes.Status {
			return nil, fmt.Errorf("failed to create paystack customer: %v", err)
		}

		wallet = models.Wallet{
			UserID:               userID,
			PaystackCustomerCode: custRes.Data.CustomerCode,
			Balance:              0,
		}
		if err := s.db.Create(&wallet).Error; err != nil {
			return nil, err
		}
	}

	return &wallet, nil
}

func (s *walletService) SetPIN(userID uuid.UUID, pin string) (*models.Wallet, error) {
	if !walletPINPattern.MatchString(pin) {
		return nil, errors.New("wallet PIN must be 4 digits")
	}
	wallet, err := s.GetOrCreateWallet(userID)
	if err != nil {
		return nil, err
	}
	if wallet.PINHash != "" {
		return nil, errors.New("wallet PIN is already set; use change PIN instead")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	wallet.PINHash = string(hash)
	wallet.PINSetAt = &now
	wallet.PINFailedAttempts = 0
	wallet.PINLockedUntil = nil
	if err := s.db.Save(wallet).Error; err != nil {
		return nil, err
	}
	return wallet, nil
}

func (s *walletService) ChangePIN(userID uuid.UUID, currentPIN string, newPIN string) (*models.Wallet, error) {
	if !walletPINPattern.MatchString(newPIN) {
		return nil, errors.New("new wallet PIN must be 4 digits")
	}
	if err := s.verifyWalletPIN(userID, currentPIN); err != nil {
		return nil, err
	}
	wallet, err := s.GetWalletByUserID(userID)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPIN), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	wallet.PINHash = string(hash)
	wallet.PINSetAt = &now
	wallet.PINFailedAttempts = 0
	wallet.PINLockedUntil = nil
	if err := s.db.Save(wallet).Error; err != nil {
		return nil, err
	}
	return wallet, nil
}

func (s *walletService) verifyWalletPIN(userID uuid.UUID, pin string) error {
	if !walletPINPattern.MatchString(pin) {
		return errors.New("enter your 4 digit wallet PIN")
	}
	var wallet models.Wallet
	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}
	if wallet.PINHash == "" {
		return errors.New("set your wallet PIN before making payments")
	}
	if wallet.PINLockedUntil != nil && wallet.PINLockedUntil.After(time.Now()) {
		return errors.New("wallet PIN is temporarily locked after too many failed attempts")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(wallet.PINHash), []byte(pin)); err != nil {
		wallet.PINFailedAttempts++
		if wallet.PINFailedAttempts >= 5 {
			lockedUntil := time.Now().Add(15 * time.Minute)
			wallet.PINLockedUntil = &lockedUntil
			wallet.PINFailedAttempts = 0
		}
		_ = s.db.Save(&wallet).Error
		return errors.New("invalid wallet PIN")
	}
	if wallet.PINFailedAttempts != 0 || wallet.PINLockedUntil != nil {
		wallet.PINFailedAttempts = 0
		wallet.PINLockedUntil = nil
		_ = s.db.Save(&wallet).Error
	}
	return nil
}

func (s *walletService) ActivateVirtualAccount(userID uuid.UUID) (*models.Wallet, error) {
	var user models.User
	if err := s.db.Preload("Profile").First(&user, userID).Error; err != nil {
		return nil, err
	}

	wallet, err := s.GetOrCreateWallet(userID)
	if err != nil {
		return nil, err
	}

	if wallet.VirtualAccountNo != "" {
		return wallet, nil
	}

	if strings.TrimSpace(user.Profile.FullName) == "" || strings.TrimSpace(user.Profile.PhoneNumber) == "" {
		return nil, errors.New("complete your full name and phone number before generating a virtual account")
	}

	names := strings.Fields(user.Profile.FullName)
	firstName := names[0]
	lastName := "User"
	if len(names) > 1 {
		lastName = names[len(names)-1]
	}

	acc, err := s.paystackService.CreateDedicatedAccount(wallet.PaystackCustomerCode, firstName, lastName, user.Profile.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual account: %v", err)
	}
	if !acc.Status || acc.Data.AccountNumber == "" {
		if acc.Message != "" {
			if acc.Meta.NextStep != "" {
				return nil, fmt.Errorf("%s. %s", acc.Message, acc.Meta.NextStep)
			}
			return nil, errors.New(acc.Message)
		}
		return nil, errors.New("paystack could not create a virtual account for this wallet")
	}

	wallet.VirtualAccountNo = acc.Data.AccountNumber
	wallet.BankName = acc.Data.Bank.Name
	wallet.AccountName = acc.Data.AccountName
	if wallet.AccountName == "" {
		wallet.AccountName = acc.Data.Assignment.AccountName
	}

	if err := s.db.Save(wallet).Error; err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *walletService) GetWalletTransactions(walletID uuid.UUID) ([]models.WalletTransaction, error) {
	var txs []models.WalletTransaction
	err := s.db.Where("wallet_id = ?", walletID).Order("created_at desc").Find(&txs).Error
	return txs, err
}

func (s *walletService) GetSmartSavingTimeline(userID uuid.UUID, propertyID uuid.UUID) (map[string]interface{}, error) {
	var profile models.Profile
	var property models.Property
	var wallet models.Wallet

	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&property, propertyID).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}

	// Logic:
	// Target = 20% of Property Price (typical downpayment)
	targetAmount := property.Price * 0.2
	if property.ListingType == models.TypeSale && property.MinimumHoldingFee > 0 {
		targetAmount = property.MinimumHoldingFee
	}

	monthlySaving := profile.MonthlyIncome * (profile.SavingPercentage / 100)

	remainingAmount := targetAmount - wallet.Balance
	if remainingAmount < 0 {
		remainingAmount = 0
	}

	monthsToGoal := 0.0
	if monthlySaving > 0 {
		monthsToGoal = remainingAmount / monthlySaving
	}

	return map[string]interface{}{
		"target_amount":    targetAmount,
		"current_balance":  wallet.Balance,
		"monthly_saving":   monthlySaving,
		"months_remaining": monthsToGoal,
		"percentage":       (wallet.Balance / targetAmount) * 100,
		"property_title":   property.Title,
	}, nil
}
func (s *walletService) Withdraw(userID uuid.UUID, amount float64, pin string) error {
	if err := s.verifyWalletPIN(userID, pin); err != nil {
		return err
	}
	var wallet models.Wallet
	var profile models.Profile

	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return err
	}

	if wallet.Balance < amount {
		return errors.New("insufficient balance")
	}

	// 1. Create Transfer Recipient
	// Note: In production, you'd use the actual bank code from a list.
	// For testing, we'll assume a bank code is provided in BankName slug.
	recipient, err := s.paystackService.CreateTransferRecipient(profile.FullName, profile.BankAccountNumber, profile.BankName)
	if err != nil || !recipient.Status {
		return errors.New("failed to create payout recipient")
	}

	// 2. Initiate Transfer on Paystack
	ref := fmt.Sprintf("wd_%s_%d", userID.String(), time.Now().Unix())
	transfer, err := s.paystackService.InitiateTransfer(amount, recipient.Data.RecipientCode, ref, "Wallet Withdrawal")
	if err != nil || !transfer.Status {
		return errors.New("failed to initiate payout")
	}

	// 3. Deduct from wallet and log transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&wallet).Update("balance", wallet.Balance-amount).Error; err != nil {
			return err
		}

		transaction := models.WalletTransaction{
			WalletID:    wallet.ID,
			Amount:      amount,
			Type:        models.TxWithdrawal,
			Status:      models.TxStatusSuccess,
			Reference:   ref,
			Description: "Payout to Bank Account",
		}
		return tx.Create(&transaction).Error
	})
}

func (s *walletService) PayReservationDeposit(userID uuid.UUID, propertyID uuid.UUID, pin string) error {
	if err := s.verifyWalletPIN(userID, pin); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&wallet).Error; err != nil {
			return err
		}

		var property models.Property
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&property, propertyID).Error; err != nil {
			return err
		}

		if property.ListingType != models.TypeSale {
			return errors.New("reservation deposits are only available for sale properties")
		}
		if property.Status != models.StatusAvailable {
			return errors.New("property is no longer available for reservation")
		}

		amount := property.MinimumHoldingFee
		if amount <= 0 {
			return errors.New("this property does not have a reservation deposit configured")
		}
		if wallet.Balance < amount {
			return errors.New("insufficient balance for reservation deposit")
		}

		ref := fmt.Sprintf("reservation_%s_%s", userID.String(), propertyID.String())
		meta, _ := json.Marshal(map[string]string{
			"property_id":  propertyID.String(),
			"payment_type": "RESERVATION_DEPOSIT",
		})
		transaction := models.WalletTransaction{
			WalletID:    wallet.ID,
			Amount:      amount,
			Type:        models.TxPayment,
			Status:      models.TxStatusSuccess,
			Reference:   ref,
			Description: fmt.Sprintf("Reservation deposit for %s", property.Title),
			MetaData:    string(meta),
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "reference"}},
			DoNothing: true,
		}).Create(&transaction)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		if err := tx.Model(&wallet).UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			return err
		}

		expiresAt := time.Now().AddDate(0, 0, 7)
		reservation := models.SaleReservation{
			BuyerID:             userID,
			PropertyID:          propertyID,
			WalletTransactionID: &transaction.ID,
			PaymentReference:    ref,
			Amount:              amount,
			Status:              models.SaleReservationReserved,
			ExpiresAt:           &expiresAt,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "payment_reference"}},
			DoNothing: true,
		}).Create(&reservation).Error; err != nil {
			return err
		}

		if err := tx.Model(&property).Update("status", models.StatusReserved).Error; err != nil {
			return err
		}

		auditDetails, _ := json.Marshal(map[string]string{
			"payment_reference": ref,
			"property_id":       propertyID.String(),
		})
		if err := tx.Create(&models.AuditEvent{
			ActorID:    &userID,
			EntityType: "SALE_RESERVATION",
			EntityID:   reservation.ID,
			Action:     "RESERVATION_DEPOSIT_PAID",
			Details:    string(auditDetails),
		}).Error; err != nil {
			return err
		}
		s.notifyReservationStakeholders(tx, property, userID)

		return nil
	})
}

func (s *walletService) notifyRentPaymentConfirmed(userID uuid.UUID, invoiceID uuid.UUID) {
	if s.mailService == nil {
		return
	}

	var invoice models.Invoice
	if err := s.db.Preload("Property").Where("id = ?", invoiceID).First(&invoice).Error; err != nil {
		log.Printf("Failed to load invoice for rent payment email: %v", err)
		return
	}
	if invoice.Type != models.InvoiceRent || invoice.Status != models.InvoicePaid {
		return
	}

	var tenant models.User
	if err := s.db.Preload("Profile").Where("id = ?", userID).First(&tenant).Error; err != nil {
		log.Printf("Failed to load tenant for rent payment email: %v", err)
		return
	}

	propertyTitle := "your property"
	if invoice.Property != nil && invoice.Property.Title != "" {
		propertyTitle = invoice.Property.Title
	}
	name := tenant.Profile.FullName
	if name == "" {
		name = "there"
	}

	go func() {
		if err := s.mailService.SendRentPaymentConfirmedEmail(tenant.Email, name, propertyTitle, invoice.Amount); err != nil {
			log.Printf("Failed to send rent payment confirmation email: %v", err)
		}
	}()
}

func (s *walletService) notifyReservationStakeholders(tx *gorm.DB, property models.Property, buyerID uuid.UUID) {
	_ = tx.Create(&models.Notification{
		UserID:  buyerID,
		Title:   "Reservation confirmed",
		Message: fmt.Sprintf("Your reservation deposit for %s has been confirmed.", property.Title),
	}).Error
	for _, recipientID := range []*uuid.UUID{property.OwnerID, property.AgentID} {
		if recipientID == nil {
			continue
		}
		_ = tx.Create(&models.Notification{
			UserID:  *recipientID,
			Title:   "Reservation payment received",
			Message: fmt.Sprintf("A buyer has paid a reservation deposit for %s.", property.Title),
		}).Error
	}
}
func (s *walletService) PayRent(userID uuid.UUID, propertyID uuid.UUID, pin string) error {
	if err := s.verifyWalletPIN(userID, pin); err != nil {
		return err
	}
	var wallet models.Wallet
	var property models.Property

	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}
	if err := s.db.First(&property, propertyID).Error; err != nil {
		return err
	}

	if property.ListingType != models.TypeRent {
		return errors.New("this property is for sale, not rent")
	}

	if property.Status != models.StatusAvailable {
		return errors.New("property is no longer available")
	}

	amount := property.Price
	if wallet.Balance < amount {
		return errors.New("insufficient balance for rent")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Deduct from Wallet
		if err := tx.Model(&wallet).Update("balance", wallet.Balance-amount).Error; err != nil {
			return err
		}

		// 2. Log Transaction
		ref := fmt.Sprintf("rent_%s_%s", userID.String(), propertyID.String())
		transaction := models.WalletTransaction{
			WalletID:    wallet.ID,
			Amount:      amount,
			Type:        models.TxPayment,
			Status:      models.TxStatusSuccess,
			Reference:   ref,
			Description: fmt.Sprintf("Annual Rent for %s", property.Title),
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		// 3. Update Property Status
		if err := tx.Model(&property).Update("status", "RENTED").Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *walletService) PayInvoice(userID uuid.UUID, invoiceID uuid.UUID, pin string) error {
	if err := s.verifyWalletPIN(userID, pin); err != nil {
		return err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&wallet).Error; err != nil {
			return err
		}

		var invoice models.Invoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Property").
			Where("id = ?", invoiceID).
			First(&invoice).Error; err != nil {
			return err
		}

		if invoice.TenantID != userID {
			return errors.New("invoice does not belong to this tenant")
		}
		if invoice.Status == models.InvoicePaid {
			return nil
		}
		if invoice.Status != models.InvoicePending && invoice.Status != models.InvoiceOverdue {
			return errors.New("only pending or overdue invoices can be paid")
		}
		if invoice.Type == models.InvoiceRent {
			if err := s.preventCrossPropertyRentPayment(tx, invoice.TenantID, invoice.PropertyID); err != nil {
				return err
			}
			if err := s.preventDuplicateRentInvoicePayment(tx, invoice); err != nil {
				return err
			}
		}
		if isStartupInvoiceType(invoice.Type) {
			if err := s.preventDuplicateStartupInvoicePayment(tx, invoice); err != nil {
				return err
			}
		}
		if wallet.Balance < invoice.Amount {
			return errors.New("insufficient wallet balance for invoice")
		}

		reference := fmt.Sprintf("inv_%s", invoice.ID.String())
		meta, _ := json.Marshal(map[string]string{
			"invoice_id":   invoice.ID.String(),
			"invoice_type": string(invoice.Type),
			"property_id":  invoice.PropertyID.String(),
		})

		description := fmt.Sprintf("%s invoice payment", invoice.Type)
		if invoice.Property != nil && invoice.Property.Title != "" {
			description = fmt.Sprintf("%s for %s", description, invoice.Property.Title)
		}

		transaction := models.WalletTransaction{
			WalletID:    wallet.ID,
			Amount:      invoice.Amount,
			Type:        models.TxPayment,
			Status:      models.TxStatusSuccess,
			Reference:   reference,
			Description: description,
			MetaData:    string(meta),
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "reference"}},
			DoNothing: true,
		}).Create(&transaction)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		now := time.Now()
		if err := tx.Model(&wallet).UpdateColumn("balance", gorm.Expr("balance - ?", invoice.Amount)).Error; err != nil {
			return err
		}

		invoice.Status = models.InvoicePaid
		invoice.PaidAt = &now
		invoice.PaymentReference = reference
		if err := tx.Save(&invoice).Error; err != nil {
			return err
		}

		if invoice.Type == models.InvoiceRent {
			if err := s.activateLeaseFromPaidRentInvoice(tx, &invoice, now); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.notifyRentPaymentConfirmed(userID, invoiceID)
	return nil
}

func (s *walletService) preventDuplicateRentInvoicePayment(tx *gorm.DB, invoice models.Invoice) error {
	if invoice.BillingPeriodStart != nil || invoice.BillingPeriodEnd != nil {
		return nil
	}

	var paidCount int64
	if err := tx.Model(&models.Invoice{}).
		Where(
			"id <> ? AND tenant_id = ? AND property_id = ? AND type = ? AND status = ? AND billing_period_start IS NULL",
			invoice.ID,
			invoice.TenantID,
			invoice.PropertyID,
			models.InvoiceRent,
			models.InvoicePaid,
		).
		Count(&paidCount).Error; err != nil {
		return err
	}
	if paidCount > 0 {
		return errors.New("the move-in rent for this property has already been paid")
	}

	return nil
}

func (s *walletService) preventDuplicateStartupInvoicePayment(tx *gorm.DB, invoice models.Invoice) error {
	var paidCount int64
	if err := tx.Model(&models.Invoice{}).
		Where(
			"id <> ? AND tenant_id = ? AND property_id = ? AND type = ? AND status = ?",
			invoice.ID,
			invoice.TenantID,
			invoice.PropertyID,
			invoice.Type,
			models.InvoicePaid,
		).
		Count(&paidCount).Error; err != nil {
		return err
	}
	if paidCount > 0 {
		return errors.New("this startup invoice has already been paid for this property")
	}

	return nil
}

func isStartupInvoiceType(invoiceType models.InvoiceType) bool {
	switch invoiceType {
	case models.InvoiceCautionDeposit,
		models.InvoiceAgencyFee,
		models.InvoiceLegalFee:
		return true
	default:
		return false
	}
}

func (s *walletService) preventCrossPropertyRentPayment(tx *gorm.DB, tenantID, propertyID uuid.UUID) error {
	var activeLease models.LeaseAgreement
	err := tx.Where("tenant_id = ? AND status = ?", tenantID, models.LeaseStatusActive).
		First(&activeLease).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if activeLease.PropertyID != propertyID {
		return errors.New("you already have an active lease on another property")
	}
	return nil
}

func (s *walletService) activateLeaseFromPaidRentInvoice(tx *gorm.DB, invoice *models.Invoice, paidAt time.Time) error {
	if invoice.LeaseID != nil {
		return s.completeRentInvoiceForExistingLease(tx, invoice)
	}

	var existingLease models.LeaseAgreement
	err := tx.Where("tenant_id = ? AND property_id = ? AND status = ?", invoice.TenantID, invoice.PropertyID, models.LeaseStatusActive).
		First(&existingLease).Error
	if err == nil {
		invoice.LeaseID = &existingLease.ID
		if err := tx.Save(invoice).Error; err != nil {
			return err
		}
		if err := s.completeRentInvoiceForExistingLease(tx, invoice); err != nil {
			return err
		}
		return tx.Model(&models.Property{}).
			Where("id = ?", invoice.PropertyID).
			Update("status", models.StatusRented).Error
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	startDate := paidAt
	endDate := startDate.AddDate(1, 0, 0)
	lease := models.LeaseAgreement{
		PropertyID:     invoice.PropertyID,
		TenantID:       invoice.TenantID,
		StartDate:      startDate,
		EndDate:        endDate,
		RentAmount:     invoice.Amount,
		TenantAccepted: false,
		Status:         models.LeaseStatusPending,
	}
	if err := tx.Create(&lease).Error; err != nil {
		return err
	}

	invoice.LeaseID = &lease.ID
	if err := tx.Save(invoice).Error; err != nil {
		return err
	}

	return tx.Model(&models.Property{}).
		Where("id = ?", invoice.PropertyID).
		Update("status", models.StatusReserved).Error
}

func (s *walletService) completeRentInvoiceForExistingLease(tx *gorm.DB, invoice *models.Invoice) error {
	if invoice.LeaseID == nil || invoice.BillingPeriodEnd == nil {
		return nil
	}

	var lease models.LeaseAgreement
	if err := tx.Where("id = ?", *invoice.LeaseID).First(&lease).Error; err != nil {
		return err
	}

	if invoice.BillingPeriodEnd.After(lease.EndDate) {
		lease.EndDate = *invoice.BillingPeriodEnd
		lease.RentAmount = invoice.Amount
		lease.Status = models.LeaseStatusActive
		if err := tx.Save(&lease).Error; err != nil {
			return err
		}
	}

	return tx.Model(&models.Property{}).
		Where("id = ?", invoice.PropertyID).
		Update("status", models.StatusRented).Error
}

func (s *walletService) DemoFund(userID uuid.UUID, amount float64) error {
	var wallet models.Wallet
	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&wallet).Update("balance", wallet.Balance+amount).Error; err != nil {
			return err
		}

		transaction := models.WalletTransaction{
			WalletID:    wallet.ID,
			Amount:      amount,
			Type:        models.TxDeposit,
			Status:      models.TxStatusSuccess,
			Reference:   fmt.Sprintf("demo_%d", time.Now().UnixNano()),
			Description: "Wallet Top-up (Demo)",
		}
		return tx.Create(&transaction).Error
	})
}
func (s *walletService) InitializePayment(userID uuid.UUID, amount float64, callbackURL string, pin string) (*payment.InitializeResponse, error) {
	if err := s.verifyWalletPIN(userID, pin); err != nil {
		return nil, err
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	reference := fmt.Sprintf("topup_%s_%d", userID.String(), time.Now().Unix())
	res, err := s.paystackService.InitializePayment(user.Email, amount, reference, callbackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize payment: %v", err)
	}
	if !res.Status {
		return nil, fmt.Errorf("failed to initialize payment: %s", res.Message)
	}

	return res, nil
}

func (s *walletService) ConfirmPayment(userID uuid.UUID, reference string) (*models.Wallet, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, errors.New("payment reference is required")
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	wallet, err := s.GetOrCreateWallet(userID)
	if err != nil {
		return nil, err
	}

	verifyRes, err := s.verifyPaystackTransactionWithRetry(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to verify payment: %v", err)
	}
	if !verifyRes.Status || verifyRes.Data.Status != "success" {
		return nil, errors.New("payment has not been confirmed by Paystack")
	}
	if verifyRes.Data.Customer.Email != user.Email {
		return nil, errors.New("payment reference does not belong to this user")
	}

	amount := verifyRes.Data.Amount / 100
	if amount <= 0 {
		return nil, errors.New("invalid payment amount")
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		transaction := models.WalletTransaction{
			WalletID:    wallet.ID,
			Amount:      amount,
			Type:        models.TxDeposit,
			Status:      models.TxStatusSuccess,
			Reference:   reference,
			Description: "Wallet Top-up via Paystack Checkout",
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "reference"}},
			DoNothing: true,
		}).Create(&transaction)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		return tx.Model(wallet).UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
	})
	if err != nil {
		return nil, err
	}

	return s.GetWalletByUserID(userID)
}

func (s *walletService) verifyPaystackTransactionWithRetry(reference string) (*payment.VerifyResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		verifyRes, err := s.paystackService.VerifyTransaction(reference)
		if err == nil {
			return verifyRes, nil
		}

		lastErr = err
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * 700 * time.Millisecond)
		}
	}

	return nil, lastErr
}
