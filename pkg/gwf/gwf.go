package gwf

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/khvh/gwf/pkg/config"
	"github.com/khvh/gwf/pkg/router"
	"github.com/khvh/gwf/pkg/util"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// App is a structure for handling application things
type App struct {
	server *fiber.App
}

// Create creates a new application instance
func Create(static embed.FS) *App {
	id := config.Get().ID

	server := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	server.Use("/docs", filesystem.New(filesystem.Config{
		Root:       http.FS(static),
		PathPrefix: "/docs",
		Browse:     false,
	}))

	prometheus := fiberprometheus.New(id)
	prometheus.RegisterAt(server, "/metrics")

	server.Use(otelfiber.Middleware(id))
	server.Use(prometheus.Middleware)
	server.Use(requestid.New())
	server.Use(recover.New())
	server.Use(cors.New())
	server.Get("/monitor", monitor.New(monitor.Config{Title: id}))

	return &App{
		server,
	}
}

func (a *App) Frontend(ui embed.FS, dir string) *App {
	if !config.Get().Server.Dev || !config.Get().Server.UI {
		return a
	}

	if config.Get().Server.Dev {
		go a.startYarnDev(dir)

		log.Trace().Msg("Frontend dev server proxy started")

		fePort := 3000

		file, err := os.ReadFile(dir + "/package.json")
		if err != nil {
			log.Trace().Err(err).Send()
		}

		var packageJson map[string]interface{}

		err = json.Unmarshal(file, &packageJson)
		if err != nil {
			log.Trace().Err(err).Send()
		} else {
			fePort = int(packageJson["devPort"].(float64))
		}

		a.server.Get("/*", func(c *fiber.Ctx) error {
			err := proxy.
				Do(c, strings.
					ReplaceAll(c.Request().URI().String(), strconv.Itoa(config.Get().Server.Port), strconv.Itoa(fePort)),
				)
			if err != nil {
				log.Err(err).Send()
			}

			return c.Send(c.Response().Body())
		})
	} else {
		return a.mountFrontend(ui, dir)
	}

	return a
}

func (a *App) mountFrontend(ui embed.FS, dir string) *App {
	a.buildYarn(dir)

	a.server.Use("/*", filesystem.New(filesystem.Config{
		Root:       http.FS(ui),
		PathPrefix: "ui/dist",
		Browse:     false,
	}))

	log.Trace().Msg("Frontend mounted")

	return a
}

func (a *App) buildYarn(dir string) {
	cmd := exec.Command("yarn", "build")

	cmd.Dir = dir

	out, err := cmd.Output()

	log.Trace().Err(err).Bytes("out", out).Send()
}

func (a *App) startYarnDev(dir string) {
	cmd := exec.Command("yarn", "dev")

	cmd.Dir = dir

	out, err := cmd.Output()

	log.Trace().Err(err).Bytes("out", out).Send()
}

// RegisterRoutes registers router.Router routes
func (a *App) RegisterRoutes(r *router.Router) *App {
	r.Build(a.server)

	return a
}

// Fiber registers routes directly with fiber
func (a *App) Fiber(fn func(app *fiber.App)) *App {
	fn(a.server)
	return a
}

// Run runs the application
func (a *App) Run() {
	id := config.Get().ID
	port := config.Get().Server.Port

	log.
		Info().
		Str("URL", fmt.Sprintf("http://0.0.0.0:%d", port)).
		Str("OpenAPI", fmt.Sprintf("http://0.0.0.0:%d/docs", port)).
		Send()

	for _, host := range util.Addresses() {
		log.
			Info().
			Str("URL", fmt.Sprintf("http://%s:%d", host, port)).
			Str("OpenAPI", fmt.Sprintf("http://%s:%d/docs", host, port)).
			Send()
	}

	log.Info().Msgf("%s started 🚀", id)

	log.Fatal().Err(a.server.Listen(fmt.Sprintf("0.0.0.0:%d", port))).Send()
}