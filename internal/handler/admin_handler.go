package handler

import (
	"strconv"
	"strings"
	"sync"

	"real-estate-backend/internal/models"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) GetSummary(c *fiber.Ctx) error {
	if h.db == nil {
		return utils.ErrorResponse(c, fiber.StatusServiceUnavailable, "Database is not configured")
	}

	var totalUsers int64
	var pendingAgents int64
	var totalProperties int64
	var activeLeases int64
	var openDisputes int64
	var openMaintenance int64
	var paymentVolume float64

	roleCounts := map[string]int64{}
	var roleRows []struct {
		Role  string
		Count int64
	}

	propertyStatusCounts := map[string]int64{}
	var propertyRows []struct {
		Status string
		Count  int64
	}

	var firstErr error
	var errMu sync.Mutex
	setErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		defer errMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	var wg sync.WaitGroup
	wg.Add(10)
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.User{}).Count(&totalUsers).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.User{}).Where("role = ? AND status = ?", models.RoleAgent, models.StatusPending).Count(&pendingAgents).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.Property{}).Count(&totalProperties).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.LeaseAgreement{}).Where("status = ?", models.LeaseStatusActive).Count(&activeLeases).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.Dispute{}).Where("status IN ?", []models.DisputeStatus{models.DisputeOpen, models.DisputeResponded, models.DisputeUnderReview}).Count(&openDisputes).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.MaintenanceRequest{}).Where("status <> ?", models.MaintenanceStatusClosed).Count(&openMaintenance).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.WalletTransaction{}).
			Where("status = ? AND type IN ?", models.TxStatusSuccess, []models.TransactionType{models.TxDeposit, models.TxPayment}).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&paymentVolume).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.User{}).Select("role, COUNT(*) as count").Group("role").Scan(&roleRows).Error)
	}()
	go func() {
		defer wg.Done()
		setErr(h.db.Model(&models.Property{}).Select("status, COUNT(*) as count").Group("status").Scan(&propertyRows).Error)
	}()

	var recentActivity []models.AuditEvent
	go func() {
		defer wg.Done()
		setErr(h.db.Preload("Actor.Profile").Order("created_at DESC").Limit(8).Find(&recentActivity).Error)
	}()
	wg.Wait()

	if firstErr != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to load admin summary")
	}

	for _, row := range roleRows {
		roleCounts[row.Role] = row.Count
	}
	for _, row := range propertyRows {
		propertyStatusCounts[row.Status] = row.Count
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Admin summary retrieved", fiber.Map{
		"total_users":            totalUsers,
		"pending_agents":         pendingAgents,
		"total_properties":       totalProperties,
		"active_leases":          activeLeases,
		"open_disputes":          openDisputes,
		"open_maintenance":       openMaintenance,
		"payment_volume":         paymentVolume,
		"role_counts":            roleCounts,
		"property_status_counts": propertyStatusCounts,
		"recent_activity":        recentActivity,
	})
}

func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	role := strings.ToUpper(strings.TrimSpace(c.Query("role")))
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Preload("Profile").Order("created_at DESC")
	if role != "" && role != "ALL" {
		query = query.Where("role = ?", role)
	}
	if status != "" && status != "ALL" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Joins("LEFT JOIN profiles ON profiles.user_id = users.id").
			Where("LOWER(users.email) LIKE ? OR LOWER(profiles.full_name) LIKE ? OR LOWER(profiles.agency_name) LIKE ?", like, like, like)
	}

	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch users")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Users retrieved", users)
}

func (h *AdminHandler) GetPendingAgents(c *fiber.Ctx) error {
	var agents []models.User
	if err := h.db.Preload("Profile").
		Where("role = ? AND status = ?", models.RoleAgent, models.StatusPending).
		Order("created_at DESC").
		Find(&agents).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch pending agents")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Pending agents retrieved", agents)
}

func (h *AdminHandler) UpdateUserStatus(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	var req struct {
		Status models.UserStatus `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}
	if req.Status != models.StatusActive && req.Status != models.StatusPending && req.Status != models.StatusSuspended {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Status must be ACTIVE, PENDING, or SUSPENDED")
	}

	var user models.User
	if err := h.db.Preload("Profile").Where("id = ?", userID).First(&user).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "User not found")
	}
	if user.Role == models.RoleSuperAdmin && req.Status != models.StatusActive {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Super admin accounts cannot be suspended from this screen")
	}

	user.Status = req.Status
	if req.Status == models.StatusSuspended {
		user.RefreshToken = ""
		user.RefreshTokenExpiry = nil
	}
	if err := h.db.Save(&user).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to update user status")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "User status updated", user)
}

func (h *AdminHandler) ListProperties(c *fiber.Ctx) error {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	listingType := strings.ToUpper(strings.TrimSpace(c.Query("listing_type")))
	search := strings.TrimSpace(c.Query("search"))
	page := 1
	pageSize := 10
	if parsed, err := strconv.Atoi(c.Query("page", "1")); err == nil && parsed > 0 {
		page = parsed
	}
	if parsed, err := strconv.Atoi(c.Query("page_size", "10")); err == nil && parsed > 0 && parsed <= 50 {
		pageSize = parsed
	}

	query := h.db.Preload("Images").
		Preload("Owner.Profile").
		Preload("Agent.Profile").
		Order("created_at DESC")

	if status != "" && status != "ALL" {
		query = query.Where("status = ?", status)
	}
	if listingType != "" && listingType != "ALL" {
		query = query.Where("listing_type = ?", listingType)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.
			Joins("LEFT JOIN users owners ON owners.id = properties.owner_id").
			Joins("LEFT JOIN profiles owner_profiles ON owner_profiles.user_id = owners.id").
			Joins("LEFT JOIN users agents ON agents.id = properties.agent_id").
			Joins("LEFT JOIN profiles agent_profiles ON agent_profiles.user_id = agents.id").
			Where(
				"LOWER(properties.title) LIKE ? OR LOWER(properties.address) LIKE ? OR LOWER(properties.city) LIKE ? OR LOWER(properties.state) LIKE ? OR LOWER(owner_profiles.full_name) LIKE ? OR LOWER(agent_profiles.full_name) LIKE ?",
				like, like, like, like, like, like,
			)
	}

	var total int64
	if err := query.Model(&models.Property{}).Count(&total).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to count properties")
	}

	var properties []models.Property
	if err := query.Limit(pageSize).Offset((page - 1) * pageSize).Find(&properties).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch properties")
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Properties retrieved", fiber.Map{
		"items":       properties,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

func (h *AdminHandler) UpdatePropertyModeration(c *fiber.Ctx) error {
	propertyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid property ID")
	}

	var req struct {
		Status     *models.PropertyStatus `json:"status"`
		IsVerified *bool                  `json:"is_verified"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	var property models.Property
	if err := h.db.Preload("Images").Preload("Owner.Profile").Preload("Agent.Profile").Where("id = ?", propertyID).First(&property).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Property not found")
	}

	if req.Status != nil {
		switch *req.Status {
		case models.StatusAvailable, models.StatusMaintenance, models.StatusCancelled, models.StatusReserved, models.StatusUnderReview, models.StatusRented, models.StatusSold:
			property.Status = *req.Status
		default:
			return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid property status")
		}
	}
	if req.IsVerified != nil {
		property.IsVerified = *req.IsVerified
	}

	if err := h.db.Save(&property).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to update property")
	}

	if err := h.db.Preload("Images").Preload("Owner.Profile").Preload("Agent.Profile").Where("id = ?", property.ID).First(&property).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to reload property")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Property moderation updated", property)
}

func (h *AdminHandler) ListPaymentLedger(c *fiber.Ctx) error {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	txType := strings.ToUpper(strings.TrimSpace(c.Query("type")))
	search := strings.TrimSpace(c.Query("search"))
	limit := 50
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	query := h.db.Preload("Wallet.User.Profile").
		Order("wallet_transactions.created_at DESC").
		Limit(limit)

	if status != "" && status != "ALL" {
		query = query.Where("wallet_transactions.status = ?", status)
	}
	if txType != "" && txType != "ALL" {
		query = query.Where("wallet_transactions.type = ?", txType)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.
			Joins("LEFT JOIN wallets ON wallets.id = wallet_transactions.wallet_id").
			Joins("LEFT JOIN users ON users.id = wallets.user_id").
			Joins("LEFT JOIN profiles ON profiles.user_id = users.id").
			Where(
				"LOWER(wallet_transactions.reference) LIKE ? OR LOWER(wallet_transactions.description) LIKE ? OR LOWER(users.email) LIKE ? OR LOWER(profiles.full_name) LIKE ?",
				like, like, like, like,
			)
	}

	var transactions []models.WalletTransaction
	if err := query.Find(&transactions).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch payment ledger")
	}

	var totalVolume float64
	h.db.Model(&models.WalletTransaction{}).
		Where("status = ?", models.TxStatusSuccess).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalVolume)

	return utils.SuccessResponse(c, fiber.StatusOK, "Payment ledger retrieved", fiber.Map{
		"transactions": transactions,
		"total_volume": totalVolume,
	})
}

func (h *AdminHandler) ListLeaseWorkflow(c *fiber.Ctx) error {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	search := strings.TrimSpace(c.Query("search"))

	requestQuery := h.db.Preload("Property").
		Preload("Tenant.Profile").
		Order("lease_requests.created_at DESC")
	if status != "" && status != "ALL" {
		requestQuery = requestQuery.Where("lease_requests.status = ?", status)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		requestQuery = requestQuery.
			Joins("LEFT JOIN properties ON properties.id = lease_requests.property_id").
			Joins("LEFT JOIN users tenants ON tenants.id = lease_requests.tenant_id").
			Joins("LEFT JOIN profiles tenant_profiles ON tenant_profiles.user_id = tenants.id").
			Where("LOWER(properties.title) LIKE ? OR LOWER(tenants.email) LIKE ? OR LOWER(tenant_profiles.full_name) LIKE ?", like, like, like)
	}

	var leaseRequests []models.LeaseRequest
	if err := requestQuery.Limit(50).Find(&leaseRequests).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch lease requests")
	}

	invoiceQuery := h.db.Preload("Property").
		Preload("Tenant.Profile").
		Preload("Lease.Documents").
		Preload("LeaseRequest").
		Where("type = ?", models.InvoiceRent).
		Order("invoices.created_at DESC")
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		invoiceQuery = invoiceQuery.
			Joins("LEFT JOIN properties ON properties.id = invoices.property_id").
			Joins("LEFT JOIN users tenants ON tenants.id = invoices.tenant_id").
			Joins("LEFT JOIN profiles tenant_profiles ON tenant_profiles.user_id = tenants.id").
			Where("LOWER(properties.title) LIKE ? OR LOWER(tenants.email) LIKE ? OR LOWER(tenant_profiles.full_name) LIKE ?", like, like, like)
	}

	var rentInvoices []models.Invoice
	if err := invoiceQuery.Limit(50).Find(&rentInvoices).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch rent invoices")
	}

	leaseQuery := h.db.Preload("Property").
		Preload("Tenant.Profile").
		Preload("Documents").
		Order("lease_agreements.created_at DESC")
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		leaseQuery = leaseQuery.
			Joins("LEFT JOIN properties ON properties.id = lease_agreements.property_id").
			Joins("LEFT JOIN users tenants ON tenants.id = lease_agreements.tenant_id").
			Joins("LEFT JOIN profiles tenant_profiles ON tenant_profiles.user_id = tenants.id").
			Where("LOWER(properties.title) LIKE ? OR LOWER(tenants.email) LIKE ? OR LOWER(tenant_profiles.full_name) LIKE ?", like, like, like)
	}

	var leases []models.LeaseAgreement
	if err := leaseQuery.Limit(50).Find(&leases).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch leases")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Lease workflow retrieved", fiber.Map{
		"lease_requests": leaseRequests,
		"rent_invoices":  rentInvoices,
		"leases":         leases,
	})
}

func (h *AdminHandler) ListSaleWorkflow(c *fiber.Ctx) error {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Preload("Buyer.Profile").
		Preload("Property").
		Preload("Documents").
		Preload("WalletTransaction").
		Order("sale_reservations.created_at DESC")

	if status != "" && status != "ALL" {
		query = query.Where("sale_reservations.status = ?", status)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.
			Joins("LEFT JOIN properties ON properties.id = sale_reservations.property_id").
			Joins("LEFT JOIN users buyers ON buyers.id = sale_reservations.buyer_id").
			Joins("LEFT JOIN profiles buyer_profiles ON buyer_profiles.user_id = buyers.id").
			Where("LOWER(properties.title) LIKE ? OR LOWER(buyers.email) LIKE ? OR LOWER(buyer_profiles.full_name) LIKE ? OR LOWER(sale_reservations.payment_reference) LIKE ?", like, like, like, like)
	}

	var reservations []models.SaleReservation
	if err := query.Limit(50).Find(&reservations).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch sale reservations")
	}

	var disputes []models.Dispute
	if err := h.db.Preload("OpenedBy.Profile").
		Preload("Respondent.Profile").
		Preload("Property").
		Preload("SaleReservation").
		Where("sale_reservation_id IS NOT NULL").
		Order("created_at DESC").
		Limit(50).
		Find(&disputes).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch sale disputes")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Sale workflow retrieved", fiber.Map{
		"reservations": reservations,
		"disputes":     disputes,
	})
}

func (h *AdminHandler) ListDisputes(c *fiber.Ctx) error {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	caseType := strings.ToUpper(strings.TrimSpace(c.Query("case_type")))
	category := strings.ToUpper(strings.TrimSpace(c.Query("category")))
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Preload("OpenedBy.Profile").
		Preload("Respondent.Profile").
		Preload("ReportedUser.Profile").
		Preload("Property").
		Preload("Invoice").
		Preload("SaleReservation").
		Preload("WalletTransaction").
		Order("disputes.created_at DESC")

	if status != "" && status != "ALL" {
		query = query.Where("disputes.status = ?", status)
	}
	if caseType != "" && caseType != "ALL" {
		query = query.Where("disputes.case_type = ?", caseType)
	}
	if category != "" && category != "ALL" {
		query = query.Where("disputes.category = ?", category)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.
			Joins("LEFT JOIN properties ON properties.id = disputes.property_id").
			Joins("LEFT JOIN users opened_by ON opened_by.id = disputes.opened_by_id").
			Joins("LEFT JOIN profiles opened_profiles ON opened_profiles.user_id = opened_by.id").
			Where(
				"LOWER(disputes.title) LIKE ? OR LOWER(disputes.reason) LIKE ? OR LOWER(disputes.description) LIKE ? OR LOWER(properties.title) LIKE ? OR LOWER(opened_by.email) LIKE ? OR LOWER(opened_profiles.full_name) LIKE ?",
				like, like, like, like, like, like,
			)
	}

	var disputes []models.Dispute
	if err := query.Limit(50).Find(&disputes).Error; err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch disputes")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Disputes retrieved", disputes)
}
