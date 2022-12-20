package router

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khvh/gwf/pkg/config"
	"github.com/khvh/gwf/pkg/spec"
	"github.com/khvh/gwf/pkg/util"
	"github.com/labstack/echo/v4"
	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"net/http"
	"runtime"
	"strings"
)

type Ctx[B interface{}, P interface{}] struct {
	Body   B
	Params P
	//Query  Q
}

// Route is a structure for holding data for building OpenAPI spec
// and handling requests with Fiber
type Route struct {
	path    string
	method  string
	spec    *spec.OAS
	handler echo.HandlerFunc
	mw      []echo.MiddlewareFunc
}

// Summary adds a summary to the route
func (r *Route) Summary(summary string) *Route {
	r.spec.AddSummary(summary)

	return r
}

// Tags add tags for the route
func (r *Route) Tags(tags ...string) *Route {
	r.spec.AddTags(tags...)

	return r
}

// Description add tags for the route
func (r *Route) Description(description string) *Route {
	r.spec.AddDescription(description)

	return r
}

// Router holds the reference for openapi3.Reflector and routes
type Router struct {
	prefix string
	group  string
	routes []*Route
}

//var (
//	lock     = &sync.Mutex{}
//	instance *Router
//)

// Instance is a singleton returning method for Router
func Instance() *Router {
	return &Router{
		routes: []*Route{},
	}
}

// InitReflector ...
func InitReflector() *openapi3.Reflector {
	conf := config.Get()
	ref := &openapi3.Reflector{}

	ref.Spec = &openapi3.Spec{Openapi: "3.0.3"}

	servers := []openapi3.Server{
		{URL: fmt.Sprintf("http://0.0.0.0:%d", conf.Server.Port)},
	}

	for _, host := range util.Addresses() {
		servers = append(servers, openapi3.Server{
			URL: fmt.Sprintf("http://%s:%d", host, conf.Server.Port),
		})
	}

	ref.Spec.WithServers(servers...)

	ref.Spec.Info.
		WithTitle(conf.OAS.Title).
		WithVersion(conf.OAS.Description).
		WithDescription(conf.OAS.Description)

	ref.SpecEns().ComponentsEns().SecuritySchemesEns().WithMapOfSecuritySchemeOrRefValuesItem(
		"bearer",
		openapi3.SecuritySchemeOrRef{
			SecurityScheme: &openapi3.SecurityScheme{
				OAuth2SecurityScheme: (&openapi3.OAuth2SecurityScheme{}).
					WithFlows(openapi3.OAuthFlows{
						Implicit: &openapi3.ImplicitOAuthFlow{
							AuthorizationURL: conf.OAuth.IssuerURL,
							Scopes:           map[string]string{},
						},
					}),
			},
		},
	)

	return ref
}

// Register registers one or more routes
func (r *Router) Register(routes ...*Route) *Router {

	for _, route := range routes {
		r.routes = append(r.routes, route)
	}

	return r
}

// Prefix adds an url prefix
func (r *Router) Prefix(url string) *Router {
	r.prefix = url

	return r
}

// Group groups routes under a common tag
func (r *Router) Group(name string) *Router {
	r.group = name

	return r
}

// Build builds the OpenAPI spec and registers handlers with Fiber
func (r *Router) Build(ref *openapi3.Reflector, app *echo.Echo) {
	for _, route := range r.routes {
		if r.prefix != "" {
			route.spec.AddPrefix(r.prefix)
		}

		if r.group != "" {
			route.spec.ReplaceTags(r.group)
		}

		route.spec.Build(ref)

		r.useRoute(route, app)
	}
}

func (r *Router) useRoute(route *Route, app *echo.Echo) {
	if r.prefix == "" {
		switch route.method {
		case http.MethodGet:
			app.GET(route.path, route.handler, route.mw...)
		case http.MethodDelete:
			app.DELETE(route.path, route.handler, route.mw...)
		case http.MethodPost:
			app.POST(route.path, route.handler, route.mw...)
		case http.MethodPut:
			app.PUT(route.path, route.handler, route.mw...)
		case http.MethodPatch:
			app.PATCH(route.path, route.handler, route.mw...)
		}
	} else {
		app := app.Group(r.prefix)

		switch route.method {
		case http.MethodGet:
			app.GET(route.path, route.handler, route.mw...)
		case http.MethodDelete:
			app.DELETE(route.path, route.handler, route.mw...)
		case http.MethodPost:
			app.POST(route.path, route.handler, route.mw...)
		case http.MethodPut:
			app.PUT(route.path, route.handler, route.mw...)
		case http.MethodPatch:
			app.PATCH(route.path, route.handler, route.mw...)
		}
	}
}

// Get creates a GET route
func Get[T interface{}](path string, handlerFunc echo.HandlerFunc, handlers ...echo.MiddlewareFunc) *Route {
	pc, _, _, _ := runtime.Caller(1)

	var t T

	return &Route{
		path:    path,
		spec:    spec.Of(path, getPackage(pc)).Get(t),
		method:  http.MethodGet,
		handler: handlerFunc,
		mw:      handlers,
	}
}

// Delete creates a DELETE route
func Delete[T interface{}](path string, handlerFunc echo.HandlerFunc, handlers ...echo.MiddlewareFunc) *Route {
	pc, _, _, _ := runtime.Caller(1)

	var t T

	return &Route{
		path:    path,
		spec:    spec.Of(path, getPackage(pc)).Delete(t),
		method:  http.MethodDelete,
		handler: handlerFunc,
		mw:      handlers,
	}
}

// Post creates a POST route
func Post[T interface{}, D interface{}](path string, handlerFunc echo.HandlerFunc, handlers ...echo.MiddlewareFunc) *Route {
	pc, _, _, _ := runtime.Caller(1)

	var (
		t T
		d D
	)

	return &Route{
		path:    path,
		spec:    spec.Of(path, getPackage(pc)).Post(t, d),
		method:  http.MethodPost,
		handler: handlerFunc,
		mw:      handlers,
	}
}

// Put creates a PUT route
func Put[T interface{}, D interface{}](path string, handlerFunc echo.HandlerFunc, handlers ...echo.MiddlewareFunc) *Route {
	pc, _, _, _ := runtime.Caller(1)

	var (
		t T
		d D
	)

	return &Route{
		path:    path,
		spec:    spec.Of(path, getPackage(pc)).Put(t, d),
		method:  http.MethodPut,
		handler: handlerFunc,
		mw:      handlers,
	}
}

// Patch creates a PATCH route
func Patch[T interface{}, D interface{}](path string, handlerFunc echo.HandlerFunc, handlers ...echo.MiddlewareFunc) *Route {
	pc, _, _, _ := runtime.Caller(1)

	var (
		t T
		d D
	)

	return &Route{
		path:    path,
		spec:    spec.Of(path, getPackage(pc)).Patch(t, d),
		method:  http.MethodPatch,
		handler: handlerFunc,
		mw:      handlers,
	}
}

// GetCtx parses and returns Ctx
func GetCtx[B interface{}, P interface{}](c *fiber.Ctx) *Ctx[B, P] {
	var (
		b B
		p P
	)

	if err := c.BodyParser(&b); err != nil {
	}

	if err := c.ParamsParser(&p); err != nil {
	}

	return &Ctx[B, P]{
		Body:   b,
		Params: p,
	}
}

func getPackage(pc uintptr) string {
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')

	if lastSlash < 0 {
		lastSlash = 0
	}

	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash

	caser := cases.Title(language.English)

	return caser.String(strings.ToLower(funcName[:lastDot]))
}