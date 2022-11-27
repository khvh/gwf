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
	return c.JSON(nil)
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
		Frontend(ui, path.Join("ui")).
		RegisterRoutes(
			router.
				Instance().
				Register(
					router.
						Get[dto.Sample]("/some/:id/path/:subId", h),
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
		Run()
}