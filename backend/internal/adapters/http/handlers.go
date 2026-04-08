package http

import (
	"net/http"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/ports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handlers bundles the driving ports for HTTP consumption.
type Handlers struct {
	authService      ports.AuthService
	bookingService   ports.BookingService
	availService     ports.AvailabilityService
	generatorService ports.SlotGeneratorService
}

func NewHandlers(
	auth ports.AuthService,
	booking ports.BookingService,
	avail ports.AvailabilityService,
	generator ports.SlotGeneratorService,
) *Handlers {
	return &Handlers{
		authService:      auth,
		bookingService:   booking,
		availService:     avail,
		generatorService: generator,
	}
}

// -----------------------------------------------------------------------------
// PUBLIC ENDPOINTS
// -----------------------------------------------------------------------------

// Login handles the admin login.
func (h *Handlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MapDomainErrorToHTTP(c, domain.NewValidationError("invalid request body"))
		return
	}

	token, user, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		MapDomainErrorToHTTP(c, err)
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: UserDetail{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

// GetAvailability returns available time slots for a specific user and date.
func (h *Handlers) GetAvailability(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		MapDomainErrorToHTTP(c, domain.NewValidationError("invalid user ID format"))
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		MapDomainErrorToHTTP(c, domain.NewValidationError("date query parameter is required (YYYY-MM-DD)"))
		return
	}

	// Parse the date. We expect the UI to ask for availability on a specific UTC day
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		MapDomainErrorToHTTP(c, domain.NewValidationError("invalid date format, expected YYYY-MM-DD"))
		return
	}

	slots, err := h.availService.GetAvailableSlots(c.Request.Context(), userID, date)
	if err != nil {
		MapDomainErrorToHTTP(c, err)
		return
	}

	// Format response
	resp := make([]TimeSlotResponse, 0, len(slots))
	for _, s := range slots {
		resp = append(resp, TimeSlotResponse{
			ID:        s.ID,
			StartTime: s.StartTimeUTC,
			EndTime:   s.EndTimeUTC,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// BookAppointment processes a reservation.
// Safe under concurrency due to Row-Level Locking in repository.
func (h *Handlers) BookAppointment(c *gin.Context) {
	var req BookAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MapDomainErrorToHTTP(c, domain.NewValidationError(err.Error())) // Provide more specific bind error
		return
	}

	// Timezone is guaranteed to be present and valid by the TimezoneMiddleware
	tz := c.GetString("client_timezone")

	domainReq := domain.BookingRequest{
		TimeSlotID:     req.TimeSlotID,
		ServiceID:      req.ServiceID,
		ClientName:     req.ClientName,
		ClientEmail:    req.ClientEmail,
		ClientTimezone: tz,
	}

	// The context passed here is wrapped by gin-contrib/timeout!
	// If the DB lock takes too long, the context will cancel.
	appointment, err := h.bookingService.Book(c.Request.Context(), domainReq)
	if err != nil {
		MapDomainErrorToHTTP(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "appointment booked successfully",
		"data": map[string]interface{}{
			"appointment_id": appointment.ID,
			"status":         appointment.Status,
		},
	})
}

// -----------------------------------------------------------------------------
// PRIVATE ENDPOINTS (Require Authentication)
// -----------------------------------------------------------------------------

// ListAppointments returns all appointments for the logged-in user.
func (h *Handlers) ListAppointments(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		MapDomainErrorToHTTP(c, domain.ErrInvalidCredentials)
		return
	}

	appointments, err := h.bookingService.ListAppointments(c.Request.Context(), userID)
	if err != nil {
		MapDomainErrorToHTTP(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": appointments})
}

// GenerateSlots creates availability slots in bulk.
func (h *Handlers) GenerateSlots(c *gin.Context) {
	tenantIDStr := c.GetString("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr)
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	var req GenerateSlotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MapDomainErrorToHTTP(c, domain.NewValidationError(err.Error()))
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)

	// Convert requested integer weekdays to time.Weekday
	var weekdays []time.Weekday
	for _, w := range req.Weekdays {
		weekdays = append(weekdays, time.Weekday(w))
	}

	domainReq := domain.SlotGenerationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		StartDate:    startDate,
		EndDate:      endDate,
		DayStartHour: req.DayStartHour,
		DayEndHour:   req.DayEndHour,
		SlotDuration: time.Duration(req.SlotDuration) * time.Minute,
		Weekdays:     weekdays,
	}

	count, err := h.generatorService.GenerateSlots(c.Request.Context(), domainReq)
	if err != nil {
		MapDomainErrorToHTTP(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "slots generated successfully",
		"data": map[string]interface{}{
			"slots_created": count,
		},
	})
}
