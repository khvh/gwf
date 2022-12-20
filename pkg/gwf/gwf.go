package gwf

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/khvh/gwf/pkg/config"
	"github.com/khvh/gwf/pkg/router"
	"github.com/khvh/gwf/pkg/util"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"github.com/swaggest/openapi-go/openapi3"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	trace "go.opentelemetry.io/otel/trace"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
)

var tracerInstance trace.Tracer

const tracerKey = "otel-go-contrib-tracer-labstack-echo"

func Tracer() trace.Tracer {
	return tracerInstance
}

// GetTracer returns trace.Tracer from request context
func GetTracer(c echo.Context) trace.Tracer {
	return c.Get("otel-go-contrib-tracer-labstack-echo").(trace.Tracer)
}

// App is a structure for handling application things
type App struct {
	server *echo.Echo
	ref    *openapi3.Reflector
}

func getFileSystem(embededFiles embed.FS) http.FileSystem {
	sub, err := fs.Sub(embededFiles, ".")
	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}

// Create creates a new application instance
func Create(static embed.FS) *App {
	id := config.Get().ID

	otel.Tracer(id)

	assetHandler := http.FileServer(getFileSystem(static))

	fmt.Println(id)

	server := echo.New()

	server.HideBanner = true
	server.HidePort = true

	server.GET("/*", echo.WrapHandler(http.StripPrefix("/", assetHandler)))

	server.Use(middleware.RequestID())
	server.Use(middleware.CORS())
	server.Use(middleware.Recover())

	prometheus.NewPrometheus(id, nil).Use(server)

	return &App{
		ref:    router.InitReflector(),
		server: server,
	}
}

func (a *App) EnableTracing() *App {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(config.Get().ID),
			),
		),
	)

	otel.
		SetTracerProvider(tp)
	otel.
		SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	a.server.Use(otelecho.Middleware(config.Get().ID))

	return a
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

		var packageJSON map[string]interface{}

		err = json.Unmarshal(file, &packageJSON)
		if err != nil {
			log.Trace().Err(err).Send()
		} else {
			fePort = int(packageJSON["devPort"].(float64))
		}

		u, _ := url.Parse("http://localhost:" + strconv.Itoa(fePort))

		a.server.Use(middleware.Proxy(middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{
			{
				URL: u,
			},
		})))
	} else {
		return a.mountFrontend(ui, dir)
	}

	return a
}

func (a *App) mountFrontend(ui embed.FS, dir string) *App {
	a.buildYarn(dir)

	//a.server.Use("/*", filesystem.New(filesystem.Config{
	//	Root:       http.FS(ui),
	//	PathPrefix: "ui/dist",
	//	Browse:     false,
	//}))

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
func (a *App) RegisterRoutes(routes ...*router.Router) *App {
	for _, r := range routes {
		r.Build(a.ref, a.server)
	}

	yamlSchema, err := a.ref.Spec.MarshalYAML()
	if err != nil {
		log.Fatal().Err(err)
	}

	jsonSchema, err := a.ref.Spec.MarshalJSON()
	if err != nil {
		log.Fatal().Err(err)
	}

	a.server.GET("/spec/spec.json", func(c echo.Context) error {
		c.Set("content-type", "application/openapi+json")

		return c.String(http.StatusOK, string(jsonSchema))
	})

	a.server.GET("/spec/spec.yaml", func(c echo.Context) error {
		c.Set("content-type", "application/openapi+yaml")

		return c.String(http.StatusOK, string(yamlSchema))
	})

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

	log.Fatal().Err(a.server.Start(fmt.Sprintf("0.0.0.0:%d", port))).Send()
}