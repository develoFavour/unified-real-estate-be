package mail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type BrevoContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

type BrevoEmailRequest struct {
	Sender      BrevoContact   `json:"sender"`
	To          []BrevoContact `json:"to"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent"`
}

type BrevoResponse struct {
	MessageID string `json:"messageId"`
}

type MailService interface {
	SendWelcomeEmail(email, name, token string) error
	SendPasswordResetEmail(email, name, token string) error
	SendAgentInvitationEmail(email, inviterName, token string) error
	SendLeaseApprovedEmail(email, name, propertyTitle string, amount float64, dueDate string) error
	SendRentPaymentConfirmedEmail(email, name, propertyTitle string, amount float64) error
	SendLeaseDocumentsReadyEmail(email, name, propertyTitle string) error
	SendLeaseActivatedEmail(email, name, propertyTitle string) error
	SendMaintenanceRequestSubmittedEmail(email, name, role, propertyTitle, requestTitle, priority string) error
	SendMaintenanceStatusUpdatedEmail(email, name, propertyTitle, requestTitle, status, note string) error
	SendMaintenanceManagerStatusUpdatedEmail(email, name, role, propertyTitle, requestTitle, status, note string) error
}

type brevoMailService struct {
	apiKey      string
	senderEmail string
	senderName  string
	appURL      string
}

func NewMailService() MailService {
	return &brevoMailService{
		apiKey:      os.Getenv("BREVO_API_KEY"),
		senderEmail: os.Getenv("SENDER_EMAIL"),
		senderName:  os.Getenv("SENDER_NAME"),
		appURL:      os.Getenv("APP_URL"),
	}
}

func (s *brevoMailService) sendEmail(to, subject, htmlContent string) error {
	if s.apiKey == "" || s.senderEmail == "" {
		return fmt.Errorf("BREVO_API_KEY and SENDER_EMAIL must be set")
	}

	payload := BrevoEmailRequest{
		Sender: BrevoContact{
			Name:  s.senderName,
			Email: s.senderEmail,
		},
		To: []BrevoContact{
			{Email: to},
		},
		Subject:     subject,
		HTMLContent: htmlContent,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling Brevo API: %v", err)
		return err
	}
	defer resp.Body.Close()

	log.Printf("Brevo API Response Status: %d", resp.StatusCode)

	var brevoResp BrevoResponse
	if err := json.NewDecoder(resp.Body).Decode(&brevoResp); err == nil {
		log.Printf("Brevo MessageID: %s", brevoResp.MessageID)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("brevo API error: %d", resp.StatusCode)
	}

	log.Printf("Email successfully queued for %s", to)
	return nil
}

func baseEmail(title, content string) string {
	return fmt.Sprintf(`
      <!DOCTYPE html>
      <html>
      <head>
        <meta charset="utf-8">
        <title>%s</title>
        <style>
          body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
          .container { max-width: 600px; margin: 0 auto; padding: 20px; }
          .header { background: #C19B76; color: black; padding: 20px; border-radius: 8px 8px 0 0; text-align: center; }
          .content { background: #f9fafb; padding: 30px; border-radius: 0 0 8px 8px; }
          .button { display: inline-block; background: #C19B76; color: black; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0; font-weight: bold; }
          .summary { background: white; border: 1px solid #eee; border-radius: 8px; padding: 16px; margin: 18px 0; }
          .footer { text-align: center; margin-top: 30px; font-size: 12px; color: #666; }
        </style>
      </head>
      <body>
        <div class="container">
          <div class="header"><h1>%s</h1></div>
          <div class="content">
            %s
            <p>Best regards,<br>The Real Estate Team</p>
          </div>
          <div class="footer"><p>This is an automated message. Please do not reply to this email.</p></div>
        </div>
      </body>
      </html>
    `, title, title, content)
}

func (s *brevoMailService) SendWelcomeEmail(email, name, token string) error {
	verificationUrl := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, token)
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>Welcome to the Real Estate Management System! We're excited to have you on board.</p>
            <p>To get started, please verify your email by clicking the button below:</p>
            <div style="text-align: center;"><a href="%s" class="button">Verify My Email</a></div>
            <p><strong>Important:</strong> This verification link will expire in 24 hours.</p>
            <p>If you didn't create this account, you can safely ignore this email.</p>
    `, name, verificationUrl)

	return s.sendEmail(email, "Verify Your Email - Real Estate Platform", baseEmail("Welcome to Our Platform", content))
}

func (s *brevoMailService) SendPasswordResetEmail(email, name, token string) error {
	resetUrl := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, token)
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>You requested to reset your password for the Real Estate Management System.</p>
            <p>Click the button below to reset your password:</p>
            <div style="text-align: center;"><a href="%s" class="button">Reset Password</a></div>
            <p><strong>Important:</strong> This reset link will expire in 1 hour.</p>
            <p>If you didn't request this password reset, you can safely ignore this email.</p>
    `, name, resetUrl)

	return s.sendEmail(email, "Reset Your Password - Real Estate Platform", baseEmail("Password Reset", content))
}

func (s *brevoMailService) SendAgentInvitationEmail(email, inviterName, token string) error {
	invitationUrl := fmt.Sprintf("%s/register?token=%s&role=AGENT", s.appURL, token)
	content := fmt.Sprintf(`
            <p>Hello,</p>
            <p><strong>%s</strong> has invited you to manage their properties on the Real Estate Management System.</p>
            <p>Join our platform as an Agent to start managing listings and earning commissions.</p>
            <div style="text-align: center;"><a href="%s" class="button">Accept Invitation</a></div>
            <p><strong>Important:</strong> This invitation will expire in 72 hours.</p>
    `, inviterName, invitationUrl)

	return s.sendEmail(email, "Invitation to Manage Property - Real Estate Platform", baseEmail("Partnership Invitation", content))
}

func (s *brevoMailService) SendLeaseApprovedEmail(email, name, propertyTitle string, amount float64, dueDate string) error {
	paymentsURL := fmt.Sprintf("%s/tenant/payments", s.appURL)
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>Your lease request for <strong>%s</strong> has been approved.</p>
            <div class="summary">
              <p><strong>Invoice Amount:</strong> NGN %.2f</p>
              <p><strong>Due Date:</strong> %s</p>
            </div>
            <p>The next step is to fund your wallet if needed and pay the rent invoice from your payments page.</p>
            <div style="text-align: center;"><a href="%s" class="button">View Invoice</a></div>
    `, name, propertyTitle, amount, dueDate, paymentsURL)

	return s.sendEmail(email, "Lease Request Approved - Rent Invoice Ready", baseEmail("Lease Request Approved", content))
}

func (s *brevoMailService) SendRentPaymentConfirmedEmail(email, name, propertyTitle string, amount float64) error {
	leasesURL := fmt.Sprintf("%s/tenant/leases", s.appURL)
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>Your rent payment of <strong>NGN %.2f</strong> for <strong>%s</strong> has been confirmed by the platform.</p>
            <p>Your lease is now pending document upload and acknowledgement. Once the owner or agent uploads the lease/handover documents, review them from your leases page.</p>
            <div style="text-align: center;"><a href="%s" class="button">View My Leases</a></div>
    `, name, amount, propertyTitle, leasesURL)

	return s.sendEmail(email, "Rent Payment Confirmed - Lease Pending Documents", baseEmail("Rent Payment Confirmed", content))
}

func (s *brevoMailService) SendLeaseDocumentsReadyEmail(email, name, propertyTitle string) error {
	leasesURL := fmt.Sprintf("%s/tenant/leases", s.appURL)
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>The lease documents for <strong>%s</strong> have been uploaded and are ready for your review.</p>
            <p>Please review the lease agreement, inspection report, handover note, and ID verification documents. If everything looks correct, acknowledge the documents from your leases page so the lease can become active.</p>
            <div style="text-align: center;"><a href="%s" class="button">Review Lease Documents</a></div>
    `, name, propertyTitle, leasesURL)

	return s.sendEmail(email, "Lease Documents Ready for Review", baseEmail("Lease Documents Ready", content))
}

func (s *brevoMailService) SendLeaseActivatedEmail(email, name, propertyTitle string) error {
	leasesURL := fmt.Sprintf("%s/tenant/leases", s.appURL)
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>Your lease for <strong>%s</strong> is now active. The property has been marked as rented on the platform.</p>
            <div style="text-align: center;"><a href="%s" class="button">View Lease</a></div>
    `, name, propertyTitle, leasesURL)

	return s.sendEmail(email, "Lease Activated - Property Rented", baseEmail("Lease Activated", content))
}

func (s *brevoMailService) SendMaintenanceRequestSubmittedEmail(email, name, role, propertyTitle, requestTitle, priority string) error {
	maintenancePath := "/owner/maintenance"
	if role == "AGENT" {
		maintenancePath = "/agent/maintenance"
	}
	maintenanceURL := fmt.Sprintf("%s%s", s.appURL, maintenancePath)
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>A new maintenance request has been submitted for <strong>%s</strong>.</p>
            <div class="summary">
              <p><strong>Request:</strong> %s</p>
              <p><strong>Priority:</strong> %s</p>
            </div>
            <p>Please review and acknowledge the request from your maintenance page.</p>
            <div style="text-align: center;"><a href="%s" class="button">View Maintenance</a></div>
    `, name, propertyTitle, requestTitle, priority, maintenanceURL)

	return s.sendEmail(email, "New Maintenance Request Submitted", baseEmail("Maintenance Request", content))
}

func (s *brevoMailService) SendMaintenanceStatusUpdatedEmail(email, name, propertyTitle, requestTitle, status, note string) error {
	maintenanceURL := fmt.Sprintf("%s/tenant/maintenance", s.appURL)
	noteHTML := ""
	if note != "" {
		noteHTML = fmt.Sprintf("<p><strong>Note:</strong> %s</p>", note)
	}
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>Your maintenance request for <strong>%s</strong> has been updated.</p>
            <div class="summary">
              <p><strong>Request:</strong> %s</p>
              <p><strong>Status:</strong> %s</p>
              %s
            </div>
            <div style="text-align: center;"><a href="%s" class="button">View Request</a></div>
    `, name, propertyTitle, requestTitle, status, noteHTML, maintenanceURL)

	return s.sendEmail(email, "Maintenance Request Updated", baseEmail("Maintenance Update", content))
}

func (s *brevoMailService) SendMaintenanceManagerStatusUpdatedEmail(email, name, role, propertyTitle, requestTitle, status, note string) error {
	maintenancePath := "/owner/maintenance"
	if role == "AGENT" {
		maintenancePath = "/agent/maintenance"
	}
	maintenanceURL := fmt.Sprintf("%s%s", s.appURL, maintenancePath)
	noteHTML := ""
	if note != "" {
		noteHTML = fmt.Sprintf("<p><strong>Note:</strong> %s</p>", note)
	}
	content := fmt.Sprintf(`
            <p>Hi %s,</p>
            <p>A maintenance request for <strong>%s</strong> has been updated by the tenant.</p>
            <div class="summary">
              <p><strong>Request:</strong> %s</p>
              <p><strong>Status:</strong> %s</p>
              %s
            </div>
            <div style="text-align: center;"><a href="%s" class="button">View Maintenance</a></div>
    `, name, propertyTitle, requestTitle, status, noteHTML, maintenanceURL)

	return s.sendEmail(email, "Maintenance Request Updated by Tenant", baseEmail("Maintenance Update", content))
}
