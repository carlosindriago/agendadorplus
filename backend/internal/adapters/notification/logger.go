package notification

import (
	"context"
	"log/slog"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/carlosindriago/agendadorplus/internal/ports"
)

// LogNotifier implements ports.Notifier by logging notification details.
// This is the MVP notification adapter — no real emails are sent.
type LogNotifier struct {
	logger *slog.Logger
}

// Compile-time check that LogNotifier implements ports.Notifier.
var _ ports.Notifier = (*LogNotifier)(nil)

// NewLogNotifier creates a new LogNotifier.
func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

// SendBookingConfirmation logs the booking confirmation details.
func (n *LogNotifier) SendBookingConfirmation(ctx context.Context, appointment *domain.Appointment) error {
	n.logger.InfoContext(ctx, "📧 BOOKING CONFIRMATION (mock)",
		"appointment_id", appointment.ID,
		"client_name", appointment.ClientName,
		"client_email", appointment.ClientEmail,
		"client_timezone", appointment.ClientTimezone,
		"status", appointment.Status,
	)
	return nil
}
