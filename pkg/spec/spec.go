package spec

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/swaggest/openapi-go/openapi3"
)

// JSONObject represents a map[string]interface{} shorthand
type JSONObject map[string]interface{}

// Error is data object for returning errors
type Error struct {
	Code           string      `json:"code,omitempty" yaml:"code,omitempty"`
	Msg            string      `json:"message,omitempty" yaml:"message,omitempty"`
	AdditionalData *JSONObject `json:"data,omitempty" yaml:"data,omitempty"`
}

// Err is the constructor for Error
func Err(code string) *Error {
	return &Error{}
}

// Message sets the message for Error
func (e *Error) Message(msg string) *Error {
	e.Msg = msg

	return e
}

// Data sets the data property for Error
func (e *Error) Data(data *JSONObject) *Error {
	e.AdditionalData = data

	return e
}

type apiResponse struct {
	code int
	body interface{}
}

// OAS is the main structure for OpenAPI generation
type OAS struct {
	path        string
	method      string
	in          interface{}
	params      []string
	headers     []string
	query       []string
	out         []*apiResponse
	tags        []string
	summary     string
	description string
}

// Of returns an instance of OAS
func Of(path string, tags ...string) *OAS {
	oas := &OAS{
		path: path,
		tags: tags,
	}

	return oas.parseParams()
}

// AddQueryParam adds query params to spec
func (o *OAS) AddQueryParam(name string) *OAS {
	o.query = append(o.query, name)

	return o
}

// AddHeaderParam adds query params to spec
func (o *OAS) AddHeaderParam(name string) *OAS {
	o.headers = append(o.headers, name)

	return o
}

// AddPrefix adds an url prefix
func (o *OAS) AddPrefix(prefix string) *OAS {
	o.path = strings.ReplaceAll(fmt.Sprintf("%s/%s", prefix, o.path), "//", "/")

	return o.parseParams()
}

// Get handles the GET request spec
func (o *OAS) Get(body interface{}, code ...int) *OAS {
	o.
		response(body, getCode(code...)).
		withNotFound().
		withInternalError().
		withMethod(http.MethodGet)

	return o
}

// Delete handles the DELETE request spec
func (o *OAS) Delete(body interface{}, code ...int) *OAS {
	o.
		response(body, getCode(code...)).
		withNotFound().
		withInternalError().
		withMethod(http.MethodDelete)

	return o
}

// Post handles the POST request spec
func (o *OAS) Post(body interface{}, data interface{}, code ...int) *OAS {
	o.
		request(data).
		response(body, getCode(code...)).
		withNotFound().
		withBadRequest().
		withInternalError().
		withMethod(http.MethodPost)

	return o
}

// Put handles the PUT request spec
func (o *OAS) Put(body interface{}, data interface{}, code ...int) *OAS {
	o.
		request(data).
		response(body, getCode(code...)).
		withNotFound().
		withBadRequest().
		withInternalError().
		withMethod(http.MethodPut)

	return o
}

// Patch handles the PATCH request spec
func (o *OAS) Patch(body interface{}, data interface{}, code ...int) *OAS {
	o.
		request(data).
		response(body, getCode(code...)).
		withNotFound().
		withBadRequest().
		withInternalError().
		withMethod(http.MethodPatch)

	return o
}

// AddSummary adds a summary for the route
func (o *OAS) AddSummary(summary string) *OAS {
	o.summary = summary

	return o
}

// AddDescription adds a description for the route
func (o *OAS) AddDescription(description string) *OAS {
	o.description = description

	return o
}

// AddTags appends tags
func (o *OAS) AddTags(tags ...string) *OAS {
	for _, tag := range tags {
		o.tags = append(o.tags, tag)
	}

	return o
}

// ReplaceTags replaces oas tags
func (o *OAS) ReplaceTags(tags ...string) *OAS {
	o.tags = tags

	return o
}

// AddResponse adds an additional response to spec
func (o *OAS) AddResponse(body interface{}, code int) *OAS {
	return o.response(body, code)
}

func (o *OAS) withNotFound() *OAS {
	return o.response(Error{}, http.StatusNotFound)
}

func (o *OAS) withBadRequest() *OAS {
	return o.response(Error{}, http.StatusBadRequest)
}

func (o *OAS) withInternalError() *OAS {
	return o.response(Error{}, http.StatusInternalServerError)
}

func (o *OAS) response(body interface{}, code int) *OAS {
	o.out = append(o.out, &apiResponse{
		code,
		body,
	})

	return o
}

func (o *OAS) withMethod(method string) *OAS {
	o.method = method

	return o
}

func (o *OAS) request(data interface{}) *OAS {
	o.in = data

	return o
}

func (o *OAS) parseParams() *OAS {
	for _, segment := range strings.Split(o.path, "/") {
		if strings.HasPrefix(segment, ":") {
			segment = strings.ReplaceAll(segment, ":", "")

			o.params = append(o.params, segment)
			o.path = strings.ReplaceAll(
				o.path,
				fmt.Sprintf(":%s", segment),
				fmt.Sprintf("{%s}", segment),
			)
		}
	}

	return o
}

func (o *OAS) createParam(id, location string) *openapi3.Parameter {
	var t openapi3.SchemaType = "string"

	param := openapi3.Parameter{}

	param.
		WithName(id).
		WithIn(openapi3.ParameterInPath).
		WithRequired(true).
		WithContentItem(id, openapi3.MediaType{
			Schema: &openapi3.SchemaOrRef{
				Schema: &openapi3.Schema{
					Title: &id,
					Type:  &t,
				},
			},
		})

	if location == "header" {
		param.
			WithIn(openapi3.ParameterInHeader).
			WithLocation(openapi3.ParameterLocation{
				HeaderParameter: &openapi3.HeaderParameter{},
			})
	}

	if location == "query" {
		param.
			WithIn(openapi3.ParameterInHeader).
			WithLocation(openapi3.ParameterLocation{
				QueryParameter: &openapi3.QueryParameter{},
			})
	}

	return &param
}

// Build constructs the OpenAPI spec for a single request
func (o *OAS) Build(ref *openapi3.Reflector) *OAS {
	op := openapi3.Operation{}

	var (
		params []openapi3.ParameterOrRef
	)

	for _, p := range o.params {
		params = append(params, openapi3.ParameterOrRef{
			Parameter: o.createParam(p, "path"),
		})
	}

	for _, q := range o.query {
		params = append(params, openapi3.ParameterOrRef{
			Parameter: o.createParam(q, "query"),
		})
	}

	for _, h := range o.headers {
		params = append(params, openapi3.ParameterOrRef{
			Parameter: o.createParam(h, "header"),
		})
	}

	op.
		WithParameters(params...).
		WithTags(o.tags...).
		WithSummary(o.summary).
		WithDescription(o.description)

	for _, response := range o.out {
		handleError(ref.SetJSONResponse(&op, response.body, response.code))
	}

	if o.method == http.MethodPost || o.method == http.MethodPut || o.method == http.MethodPatch {
		handleError(ref.SetRequest(&op, o.in, o.method))
	}

	handleError(ref.Spec.AddOperation(o.method, o.path, op))

	return o
}

func handleError(err error) {
	if err != nil {
		log.Print(err)
	}
}

func getCode(code ...int) int {
	statusCode := 200

	if len(code) > 0 {
		statusCode = code[0]
	}

	return statusCode
}
