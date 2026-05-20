package models

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string
type TransactionStatus string

const (
	TxDeposit    TransactionType = "DEPOSIT"
	TxWithdrawal TransactionType = "WITHDRAWAL"
	TxPayment    TransactionType = "PAYMENT"
	TxRefund     TransactionType = "REFUND"

	TxStatusPending   TransactionStatus = "PENDING"
	TxStatusSuccess   TransactionStatus = "SUCCESS"
	TxStatusFailed    TransactionStatus = "FAILED"
	TxStatusCancelled TransactionStatus = "CANCELLED"
)

type Wallet struct {
	ID                   uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID               uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Balance              float64    `gorm:"type:numeric(12,2);default:0" json:"balance"`
	Currency             string     `gorm:"type:varchar(5);default:'NGN'" json:"currency"`
	PaystackCustomerCode string     `gorm:"type:varchar(100)" json:"paystack_customer_code"`
	VirtualAccountNo     string     `gorm:"type:varchar(20)" json:"virtual_account_no"`
	BankName             string     `gorm:"type:varchar(100)" json:"bank_name"`
	AccountName          string     `gorm:"type:varchar(255)" json:"account_name"`
	PINHash              string     `gorm:"type:varchar(255)" json:"-"`
	PINSetAt             *time.Time `json:"pin_set_at"`
	PINFailedAttempts    int        `gorm:"default:0" json:"-"`
	PINLockedUntil       *time.Time `json:"pin_locked_until"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type WalletTransaction struct {
	ID          uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	WalletID    uuid.UUID         `gorm:"type:uuid;index;not null" json:"wallet_id"`
	Amount      float64           `gorm:"type:numeric(12,2);not null" json:"amount"`
	Type        TransactionType   `gorm:"type:varchar(20);not null;index:idx_wallet_transactions_type_status_created,priority:1" json:"type"`
	Status      TransactionStatus `gorm:"type:varchar(20);default:'PENDING';index:idx_wallet_transactions_type_status_created,priority:2" json:"status"`
	Reference   string            `gorm:"type:varchar(100);uniqueIndex" json:"reference"`
	Description string            `gorm:"type:text" json:"description"`
	MetaData    string            `gorm:"type:text" json:"meta_data"` // JSON string for extra info
	CreatedAt   time.Time         `gorm:"index;index:idx_wallet_transactions_type_status_created,priority:3" json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`

	Wallet *Wallet `gorm:"foreignKey:WalletID" json:"wallet,omitempty"`
}
