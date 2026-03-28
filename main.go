package main

import (
	"helloworld/db"
	"helloworld/logs"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	zlog "github.com/rs/zerolog/log"
)

const (
	certDir = "cert"
	// Standard PEM filenames; place your certificate chain and private key here.
	certFile = "cert.pem"
	keyFile  = "key.pem"
)

// HTTPS default; use 443 in production if you have permission to bind a privileged port.
var listenAddr = ":3000"

func main() {
	if err := logs.Init(); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = logs.Close() }()

	db.InitDB()

	zlog.Info().Msg("server starting")

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		StrictRouting: true,
		ServerHeader:  "Fiber",
		AppName:       "Test App v1.0.1",
	})

	app.Use(logger.New(logs.FiberMiddleware()))

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	// certPath := filepath.Join(certDir, certFile)
	// keyPath := filepath.Join(certDir, keyFile)

	// if err := app.Listen(listenAddr, fiber.ListenConfig{
	// 	CertFile:    certPath,
	// 	CertKeyFile: keyPath,
	// }); err != nil {
	// 	log.Fatal(err)
	// }

	app.Listen(listenAddr)
}
