package main

import (
	"embed"
	"github.com/gofiber/fiber/v2"
	"github.com/khvh/gwf/pkg/config"
	"github.com/khvh/gwf/pkg/core/dto"
	"github.com/khvh/gwf/pkg/gwf"
	"github.com/khvh/gwf/pkg/logger"
	"github.com/khvh/gwf/pkg/router"
	"path"
)

func h(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": true})
}

//go:embed docs/*
var content embed.FS

//go:embed ui/dist/*
var ui embed.FS

func main() {
	if err := config.Autoload(); err != nil {
		panic(err)
	}

	logger.Init(config.Get().Server.Dev)

	gwf.
		Create(content).
		RegisterRoutes(
			router.
				Instance().
				Group("tests").
				Prefix("/api/v1").
				Register(
					router.
						Get[dto.Sample]("/some/:id/path/:subId", h).Summary("Testing summary").Description("kek"),
					router.
						Delete[dto.Sample]("/some/:id/path/:subId", h),
					router.
						Post[dto.Sample, dto.Sample]("/some/:id/path", h),
					router.
						Put[dto.Sample, dto.Sample]("/some/:id/path/:subId", h),
					router.
						Patch[dto.Sample, dto.Sample]("/some/:id/path/:subId", h),
				),
		).
		Fiber(func(app *fiber.App) {
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.JSON(true)
			})
		}).
		Frontend(ui, path.Join("ui")).
		Run()
}