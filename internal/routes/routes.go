package routes

import (
	"log"
	"real-estate-backend/internal/config"
	"real-estate-backend/internal/handler"
	"real-estate-backend/internal/middleware"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/mail"
	"real-estate-backend/pkg/payment"
	"real-estate-backend/pkg/upload"
	websocketpkg "real-estate-backend/pkg/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, cfg *config.Config) {
	// Initialize Mail Service
	mailService := mail.NewMailService()

	// Initialize Repositories
	userRepo := repository.NewUserRepository(db)
	propRepo := repository.NewPropertyRepository(db)
	invitationRepo := repository.NewInvitationRepository(db)
	leaseRepo := repository.NewLeaseRepository(db)
	maintenanceRepo := repository.NewMaintenanceRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	leaseRequestRepo := repository.NewLeaseRequestRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	paystackService := payment.NewPaystackService(cfg.PaystackSecretKey)

	// Initialize Hub
	hub := websocketpkg.NewHub()
	go hub.Run()

	// Initialize Services
	authService := service.NewAuthService(userRepo, invitationRepo, propRepo, mailService)
	propertyService := service.NewPropertyService(propRepo, userRepo, invitationRepo, maintenanceRepo, paymentRepo, mailService)
	maintenanceService := service.NewMaintenanceService(maintenanceRepo, leaseRepo, propRepo, userRepo, mailService)
	agentService := service.NewAgentService(propRepo, invitationRepo, maintenanceRepo)
	tenantService := service.NewTenantService(leaseRepo, propRepo, maintenanceRepo, paymentRepo, db)
	paymentService := service.NewPaymentService(paymentRepo)
	messageService := service.NewMessageService(messageRepo, hub)
	walletService := service.NewWalletService(db, paystackService, mailService)
	invoiceService := service.NewInvoiceService(invoiceRepo, db, notificationRepo)
	leaseRequestService := service.NewLeaseRequestService(leaseRequestRepo, propRepo, leaseRepo, invoiceService, notificationRepo, mailService)
	riskWorkflowService := service.NewRiskWorkflowService(db, mailService)
	invoiceService.StartRecurringBillingScheduler()

	// Initialize Handlers
	authHandler := handler.NewAuthHandler(authService, cfg)
	propertyHandler := handler.NewPropertyHandler(propertyService)
	maintenanceHandler := handler.NewMaintenanceHandler(maintenanceService)
	agentHandler := handler.NewAgentHandler(agentService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	messageHandler := handler.NewMessageHandler(messageService, hub, cfg.JWTSecret)
	walletHandler := handler.NewWalletHandler(walletService)
	webhookHandler := handler.NewWebhookHandler(db, cfg)
	leaseRequestHandler := handler.NewLeaseRequestHandler(leaseRequestService)
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)
	riskWorkflowHandler := handler.NewRiskWorkflowHandler(riskWorkflowService)
	adminHandler := handler.NewAdminHandler(db)

	api := app.Group("/api/v1")

	// WebSocket Route (Top level for reliability)
	app.Get("/ws", websocket.New(messageHandler.HandleWS, websocket.Config{
		Origins: []string{"*"},
	}))

	// Webhooks (Public)
	api.Post("/webhooks/paystack", webhookHandler.PaystackWebhook)

	// Auth Routes
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/verify-email", authHandler.VerifyEmail)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/reset-password", authHandler.ResetPassword)
	auth.Get("/me", middleware.RequireAuth(cfg.JWTSecret), authHandler.GetMe)
	auth.Put("/update-profile", middleware.RequireAuth(cfg.JWTSecret), authHandler.UpdateProfile)
	auth.Get("/invitation/:token", authHandler.ValidateInvitation)
	auth.Get("/agents/verified", middleware.RequireAuth(cfg.JWTSecret), authHandler.GetVerifiedAgents)

	// Admin Routes
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAuth(cfg.JWTSecret))
	admin.Use(middleware.RequireRole(string(models.RoleSuperAdmin)))
	admin.Get("/summary", adminHandler.GetSummary)
	admin.Get("/users", adminHandler.ListUsers)
	admin.Get("/users/pending-agents", adminHandler.GetPendingAgents)
	admin.Patch("/users/:id/status", adminHandler.UpdateUserStatus)
	admin.Get("/properties", adminHandler.ListProperties)
	admin.Patch("/properties/:id/moderation", adminHandler.UpdatePropertyModeration)
	admin.Get("/payments", adminHandler.ListPaymentLedger)
	admin.Get("/leases", adminHandler.ListLeaseWorkflow)
	admin.Get("/sales", adminHandler.ListSaleWorkflow)
	admin.Get("/disputes", adminHandler.ListDisputes)

	// Property Routes
	props := api.Group("/properties")
	props.Get("/", propertyHandler.GetAllProperties)
	props.Get("/owned", middleware.RequireAuth(cfg.JWTSecret), propertyHandler.GetOwnedProperties)
	props.Get("/:id", propertyHandler.GetProperty)

	// Protected Property Routes
	props.Use(middleware.RequireAuth(cfg.JWTSecret))
	props.Post("/", propertyHandler.CreateProperty)
	props.Get("/me/all", propertyHandler.GetMyProperties)
	props.Get("/owner/agents", propertyHandler.GetOwnerAgents)
	props.Get("/owner/summary", propertyHandler.GetOwnerSummary)
	props.Post("/:id/assign-agent", propertyHandler.AssignAgent)
	props.Patch("/:id/sale-status", propertyHandler.UpdateSaleStatus)

	// Upload Routes
	uploadService, err := upload.NewUploadService()
	if err != nil {
		log.Printf("WARNING: Upload Service failed to initialize: %v", err)
	}
	uploadHandler := handler.NewUploadHandler(uploadService)
	upload := api.Group("/upload")
	upload.Use(middleware.RequireAuth(cfg.JWTSecret))
	upload.Post("/image", uploadHandler.UploadImage)
	upload.Post("/document", uploadHandler.UploadDocument)

	// Maintenance Routes
	maint := api.Group("/maintenance")
	maint.Use(middleware.RequireAuth(cfg.JWTSecret))
	maint.Get("/owner", maintenanceHandler.GetOwnerRequests)
	maint.Get("/agent", maintenanceHandler.GetAgentRequests)
	maint.Post("/", maintenanceHandler.CreateRequest)
	maint.Patch("/:id/status", maintenanceHandler.UpdateStatus)
	maint.Post("/:id/close", maintenanceHandler.CloseRequest)
	maint.Post("/:id/reopen", maintenanceHandler.ReopenRequest)

	// Payment Routes
	payments := api.Group("/payments")
	payments.Use(middleware.RequireAuth(cfg.JWTSecret))
	payments.Get("/owner", paymentHandler.GetOwnerIncome)

	// Agent Routes
	agent := api.Group("/agent")
	agent.Use(middleware.RequireAuth(cfg.JWTSecret))
	agent.Use(middleware.RequireRole(string(models.RoleAgent)))
	agent.Get("/summary", agentHandler.GetDashboardSummary)
	agent.Post("/invitations/:id/accept", agentHandler.AcceptInvitation)
	agent.Post("/mandates/:id/accept", agentHandler.AcceptMandate)

	// Tenant Routes
	tenant := api.Group("/tenant")
	tenant.Use(middleware.RequireAuth(cfg.JWTSecret))
	tenant.Get("/dashboard", tenantHandler.GetDashboard)
	tenant.Get("/payments", tenantHandler.GetPayments)
	tenant.Post("/lease-requests/:propertyID", leaseRequestHandler.CreateRequest)
	tenant.Get("/lease-requests", leaseRequestHandler.GetMyRequests)
	tenant.Get("/invoices", invoiceHandler.GetMyInvoices)

	// Lease Request Routes
	leaseRequests := api.Group("/lease-requests")
	leaseRequests.Use(middleware.RequireAuth(cfg.JWTSecret))
	leaseRequests.Get("/incoming", leaseRequestHandler.GetIncomingRequests)
	leaseRequests.Patch("/:id/status", leaseRequestHandler.ReviewRequest)

	// Invoice Routes
	invoices := api.Group("/invoices")
	invoices.Use(middleware.RequireAuth(cfg.JWTSecret))
	invoices.Get("/incoming", invoiceHandler.GetIncomingInvoices)
	invoices.Post("/", invoiceHandler.CreateManualInvoice)
	invoices.Post("/billing/run", invoiceHandler.RunBillingCycle)

	// Sale Reservation, Evidence, and Dispute Routes
	reservations := api.Group("/sale-reservations")
	reservations.Use(middleware.RequireAuth(cfg.JWTSecret))
	reservations.Get("/mine", riskWorkflowHandler.GetMySaleReservations)
	reservations.Get("/incoming", riskWorkflowHandler.GetIncomingSaleReservations)
	reservations.Post("/expiry/run", riskWorkflowHandler.RunReservationExpiryCheck)
	reservations.Post("/:id/accept-documents", riskWorkflowHandler.AcceptSaleDocuments)
	reservations.Post("/:id/settlement", riskWorkflowHandler.RecordFinalSettlement)

	documents := api.Group("/transaction-documents")
	documents.Use(middleware.RequireAuth(cfg.JWTSecret))
	documents.Get("/", riskWorkflowHandler.GetDocuments)
	documents.Post("/", riskWorkflowHandler.UploadDocument)

	leases := api.Group("/leases")
	leases.Use(middleware.RequireAuth(cfg.JWTSecret))
	leases.Post("/:id/accept-documents", riskWorkflowHandler.AcceptLeaseDocuments)

	disputes := api.Group("/disputes")
	disputes.Use(middleware.RequireAuth(cfg.JWTSecret))
	disputes.Post("/", riskWorkflowHandler.OpenDispute)
	disputes.Get("/mine", riskWorkflowHandler.GetMyDisputes)
	disputes.Patch("/:id/respond", riskWorkflowHandler.RespondToDispute)
	disputes.Patch("/:id/resolve", riskWorkflowHandler.ResolveDispute)

	// Messaging Routes
	msg := api.Group("/messages")
	msg.Use(middleware.RequireAuth(cfg.JWTSecret))
	msg.Get("/conversations", messageHandler.GetConversations)
	msg.Get("/history/:id", messageHandler.GetChatHistory)
	msg.Post("/send", messageHandler.SendMessage)

	// Wallet Routes
	wallet := api.Group("/wallet")
	wallet.Use(middleware.RequireAuth(cfg.JWTSecret))
	wallet.Get("/", walletHandler.GetMyWallet)
	wallet.Get("/transactions", walletHandler.GetTransactions)
	wallet.Get("/saving-timeline", walletHandler.GetSmartSavingTimeline)
	wallet.Post("/withdraw", walletHandler.Withdraw)
	wallet.Post("/pay-holding-fee/:propertyID", walletHandler.PayHoldingFee)
	wallet.Post("/pay-rent/:propertyID", walletHandler.PayRent)
	wallet.Post("/virtual-account", walletHandler.ActivateVirtualAccount)
	wallet.Post("/pin", walletHandler.SetPIN)
	wallet.Patch("/pin", walletHandler.ChangePIN)
	wallet.Post("/initialize", walletHandler.InitializePayment)
	wallet.Post("/confirm", walletHandler.ConfirmPayment)
	wallet.Post("/invoices/:invoiceID/pay", walletHandler.PayInvoice)
	wallet.Post("/fund-demo", walletHandler.DemoFund)

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "message": "Server is running"})
	})
}
