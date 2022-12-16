package main

import (
	"embed"
	"fmt"
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

func p(c *fiber.Ctx) error {
	asd := router.GetCtx[dto.Sample, dto.Sample](c)

	fmt.Println(asd.Body.ID)

	return c.JSON(asd)
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
				Group("asd").
				Prefix("/api/v1").
				Register(
					router.
						Get[dto.Sample]("/some/:id/path/:subId", h).Summary("Testing summary").Description("kek").Tags("1"),
					router.
						Delete[dto.Sample]("/some/:id/path/:subId", h).Tags("1"),
					router.
						Post[dto.Sample, dto.Sample]("/some/:id/path", p).Tags("1"),
					router.
						Put[dto.Sample, dto.Sample]("/some/:id/path/:subId", h).Tags("1"),
					router.
						Patch[dto.Sample, dto.Sample]("/some/:id/path/:subId", h).Tags("1"),
				),

			router.
				Instance().
				Group("asd2").
				Prefix("/api/v2").
				Register(
					router.
						Get[dto.Sample]("/2some/:id/path/:subId", h).Summary("Testing summary").Description("kek").Tags("2"),
					router.
						Delete[dto.Sample]("/2some/:id/path/:subId", h).Tags("2"),
					router.
						Post[dto.Sample, dto.Sample]("/2some/:id/path", p).Tags("2"),
					router.
						Put[dto.Sample, dto.Sample]("/2some/:id/path/:subId", h).Tags("2"),
					router.
						Patch[dto.Sample, dto.Sample]("/2some/:id/path/:subId", h).Tags("2"),
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