package spec

import (
	"fmt"
	"github.com/swaggest/openapi-go/openapi3"
	"log"
	"net/http"
	"strings"
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
	path   string
	method string
	in     interface{}
	params []string
	out    []*apiResponse
	tags   []string
}

// Of returns an instance of OAS
func Of(path string, tags ...string) *OAS {
	oas := &OAS{
		path: path,
		tags: tags,
	}

	return oas.parseParams()
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

// Build constructs the OpenAPI spec for a single request
func (o *OAS) Build(ref *openapi3.Reflector) *OAS {
	op := openapi3.Operation{}

	var (
		params []openapi3.ParameterOrRef
	)

	for _, p := range o.params {
		var t openapi3.SchemaType

		t = "string"

		param := openapi3.Parameter{}

		param.
			WithName(p).
			WithIn(openapi3.ParameterInPath).
			WithRequired(true).
			WithContentItem("id", openapi3.MediaType{
				Schema: &openapi3.SchemaOrRef{
					Schema: &openapi3.Schema{
						Title: &p,
						Type:  &t,
					},
				},
			})

		params = append(params, openapi3.ParameterOrRef{
			Parameter: &param,
		})
	}

	op.WithParameters(params...)

	op.WithTags(o.tags...)

	for _, response := range o.out {
		handleError(ref.SetJSONResponse(&op, response.body, response.code))
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