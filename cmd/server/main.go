package main

import (
	"flag"
	"log"

	"real-estate-backend/internal/config"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/routes"
	"real-estate-backend/pkg/database"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Parse flags
	migrate := flag.Bool("migrate", false, "Run database migrations")
	flag.Parse()

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations if flag is set
	if *migrate {
		if err := models.AutoMigrate(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migration successful, exiting.")
		return
	}

	if db == nil {
		log.Println("Running in offline mode (no DB connection)")
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Upgrade, Connection",
		AllowCredentials: true,
	}))

	// Register Routes
	routes.SetupRoutes(app, db, cfg)

	// Start server
	log.Printf("Server listening on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
