package http

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures the Gin engine.
func SetupRouter(
	handlers *Handlers,
	jwtSecret string,
	reqTimeout time.Duration,
	corsOrigin string,
) *gin.Engine {
	// Use default Gin setup (includes Logger and Recovery middleware)
	r := gin.Default()

	// CORS Setup
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{corsOrigin}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Timezone"}
	r.Use(cors.New(config))

	// API Group setup
	api := r.Group("/api/v1")

	// -------------------------------------------------------------------------
	// PUBLIC ROUTES
	// -------------------------------------------------------------------------
	public := api.Group("/")
	public.Use(TimezoneMiddleware()) // All public endpoints can provide timezone
	
	// Authentication
	public.POST("/auth/login", handlers.Login)

	// Booking Flow (Public)
	public.GET("/availability/:userId", handlers.GetAvailability)
	
	// Complex transaction route: protected by strict timeout via middleware
	public.POST("/appointments", TimeoutMiddleware(reqTimeout), handlers.BookAppointment)

	// -------------------------------------------------------------------------
	// PRIVATE ROUTES (Admin Panel)
	// -------------------------------------------------------------------------
	private := api.Group("/")
	private.Use(AuthMiddleware(jwtSecret)) // JWT check

	private.GET("/appointments", handlers.ListAppointments)
	private.POST("/slots/generate", handlers.GenerateSlots)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}
