package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type PaystackService interface {
	VerifyTransaction(reference string) (*VerifyResponse, error)
	InitializePayment(email string, amount float64, reference string, callbackURL string) (*InitializeResponse, error)
	CreateCustomer(email, firstName, lastName, phone string) (*CustomerResponse, error)
	CreateDedicatedAccount(customerID, firstName, lastName, phone string) (*DedicatedAccountResponse, error)
	InitiateTransfer(amount float64, recipientCode, reference, reason string) (*TransferResponse, error)
	CreateTransferRecipient(name, accountNumber, bankCode string) (*RecipientResponse, error)
}

type paystackService struct {
	secretKey string
}

func NewPaystackService(secretKey string) PaystackService {
	return &paystackService{secretKey}
}

type VerifyResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Amount          float64 `json:"amount"`
		Status          string  `json:"status"`
		Reference       string  `json:"reference"`
		GatewayResponse string  `json:"gateway_response"`
		Customer        struct {
			Email string `json:"email"`
		} `json:"customer"`
	} `json:"data"`
}

type CustomerResponse struct {
	Status bool `json:"status"`
	Data   struct {
		CustomerCode string `json:"customer_code"`
		ID           int    `json:"id"`
	} `json:"data"`
}

type DedicatedAccountResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Meta    struct {
		NextStep string `json:"nextStep"`
	} `json:"meta"`
	Data struct {
		Bank struct {
			Name string `json:"name"`
		} `json:"bank"`
		AccountNumber string `json:"account_number"`
		AccountName   string `json:"account_name"`
		Assignment    struct {
			AccountName string `json:"account_name"`
		} `json:"assignment"`
	} `json:"data"`
}

type InitializeResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

type RecipientResponse struct {
	Status bool `json:"status"`
	Data   struct {
		RecipientCode string `json:"recipient_code"`
	} `json:"data"`
}

type TransferResponse struct {
	Status bool `json:"status"`
	Data   struct {
		Reference    string  `json:"reference"`
		Amount       float64 `json:"amount"`
		TransferCode string  `json:"transfer_code"`
		Status       string  `json:"status"`
	} `json:"data"`
}

func (p *paystackService) postRequest(url string, payload interface{}, target interface{}) error {
	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+p.secretKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	fmt.Printf("PAYSTACK DEBUG: URL=%s | BODY=%s\n", url, string(body))
	return json.Unmarshal(body, target)
}

func (p *paystackService) InitializePayment(email string, amount float64, reference string, callbackURL string) (*InitializeResponse, error) {
	url := "https://api.paystack.co/transaction/initialize"
	payload := map[string]interface{}{
		"email":     email,
		"amount":    amount * 100, // kobo
		"reference": reference,
	}
	if callbackURL != "" {
		payload["callback_url"] = callbackURL
	}
	var res InitializeResponse
	err := p.postRequest(url, payload, &res)
	return &res, err
}

func (p *paystackService) VerifyTransaction(reference string) (*VerifyResponse, error) {
	url := fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+p.secretKey)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var verifyRes VerifyResponse
	body, _ := io.ReadAll(res.Body)
	json.Unmarshal(body, &verifyRes)
	return &verifyRes, nil
}

func (p *paystackService) CreateCustomer(email, firstName, lastName, phone string) (*CustomerResponse, error) {
	url := "https://api.paystack.co/customer"
	payload := map[string]string{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"phone":      phone,
	}
	var res CustomerResponse
	err := p.postRequest(url, payload, &res)
	return &res, err
}

func (p *paystackService) CreateDedicatedAccount(customerID, firstName, lastName, phone string) (*DedicatedAccountResponse, error) {
	url := "https://api.paystack.co/dedicated_account"
	preferredBank := "wema-bank"
	if strings.HasPrefix(p.secretKey, "sk_test_") {
		preferredBank = "test-bank"
	}

	payload := map[string]string{
		"customer":       customerID,
		"preferred_bank": preferredBank,
		"first_name":     firstName,
		"last_name":      lastName,
		"phone":          phone,
	}
	var res DedicatedAccountResponse
	err := p.postRequest(url, payload, &res)
	return &res, err
}

func (p *paystackService) CreateTransferRecipient(name, accountNumber, bankCode string) (*RecipientResponse, error) {
	url := "https://api.paystack.co/transferrecipient"
	payload := map[string]string{
		"type":           "nuban",
		"name":           name,
		"account_number": accountNumber,
		"bank_code":      bankCode,
		"currency":       "NGN",
	}
	var res RecipientResponse
	err := p.postRequest(url, payload, &res)
	return &res, err
}

func (p *paystackService) InitiateTransfer(amount float64, recipientCode, reference, reason string) (*TransferResponse, error) {
	url := "https://api.paystack.co/transfer"
	payload := map[string]interface{}{
		"source":    "balance",
		"amount":    amount * 100, // Paystack uses kobo
		"recipient": recipientCode,
		"reference": reference,
		"reason":    reason,
	}
	var res TransferResponse
	err := p.postRequest(url, payload, &res)
	return &res, err
}
