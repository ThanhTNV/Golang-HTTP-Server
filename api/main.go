package main

import (
	"helloworld/db"
	"helloworld/logs"
	"log"

	loghandler "helloworld/app/logs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	zlog "github.com/rs/zerolog/log"
)

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

	loghandler.RegisterRoutes(app)

	app.Listen(listenAddr)
}
