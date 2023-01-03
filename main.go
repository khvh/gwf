package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"github.com/khvh/gwf/pkg/config"
	"github.com/khvh/gwf/pkg/core/dto"
	"github.com/khvh/gwf/pkg/gwf"
	"github.com/khvh/gwf/pkg/logger"
	"github.com/khvh/gwf/pkg/queue"
	"github.com/khvh/gwf/pkg/router"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"time"
)

func h(c echo.Context) error {
	return c.JSON(200, map[string]bool{"status": true})
}

//go:embed docs
var content embed.FS

//go:embed ui/dist/*
var ui embed.FS

const (
	TypeEmailDelivery = "email:deliver"
	TypeImageResize   = "image:resize"
)

type EmailDeliveryPayload struct {
	UserID     int
	TemplateID string
}

type ImageResizePayload struct {
	SourceURL string
}

//----------------------------------------------
// Write a function NewXXXTask to create a task.
// A task consists of a type and a payload.
//----------------------------------------------

func NewEmailDeliveryTask(userID int, tmplID string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailDeliveryPayload{UserID: userID, TemplateID: tmplID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEmailDelivery, payload), nil
}

func NewImageResizeTask(src string) (*asynq.Task, error) {
	payload, err := json.Marshal(ImageResizePayload{SourceURL: src})
	if err != nil {
		return nil, err
	}
	// task options can be passed to NewTask, which can be overridden at enqueue time.
	return asynq.NewTask(TypeImageResize, payload, asynq.MaxRetry(5), asynq.Timeout(20*time.Minute)), nil
}

//---------------------------------------------------------------
// Write a function HandleXXXTask to handle the input task.
// Note that it satisfies the asynq.HandlerFunc interface.
//
// Handler doesn't need to be a function. You can define a type
// that satisfies asynq.Handler interface. See examples below.
//---------------------------------------------------------------

func HandleEmailDeliveryTask(ctx context.Context, t *asynq.Task) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	log.Trace().Msgf("Sending Email to User: user_id=%d, template_id=%s", p.UserID, p.TemplateID)
	// Email delivery code ...
	return nil
}

// ImageProcessor implements asynq.Handler interface.
type ImageProcessor struct {
	// ... fields for struct
}

func (processor *ImageProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ImageResizePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	log.Trace().Msgf("Resizing image: src=%s", p.SourceURL)
	// Image resizing code ...
	return nil
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

func main() {
	if err := config.Autoload(); err != nil {
		panic(err)
	}

	logger.Init(config.Get().Server.Dev)

	gwf.
		Create(content).
		EnableTracing().
		Configure(func(e *echo.Echo) {
			e.GET("/runtask", func(c echo.Context) error {
				task, err := NewEmailDeliveryTask(42, "some:template:id")
				if err != nil {
					log.Fatal().Err(err)
				}

				queue.NewClient("127.0.0.1:6379").Add(task)

				return c.JSON(200, true)
			})
		}).
		RegisterRoutes(
			router.
				Instance().
				Group("asd").
				Prefix("/api/v1").
				Register(
					router.Get[dto.Sample]("", h),
					router.
						Get[dto.Sample]("/some/:id/path/:subId", h).Query("lol").Header("lmao").Summary("Testing summary").Description("kek").Tags("1"),
					router.
						Delete[dto.Sample]("/some/:id/path/:subId", h).Tags("1"),
					router.
						Post[dto.Sample, dto.Sample]("/some/:id/path", h).Tags("1").Query("lol"),
					router.
						Put[dto.Sample, dto.Sample]("/some/:id/path/:subId", h).Tags("1"),
					router.
						Patch[dto.Sample, dto.Sample]("/some/:id/path/:subId", h).Tags("1"),
				),
		).
		Queue(func(q *queue.Queue) {
			q.
				AddHandlerFunc(TypeEmailDelivery, HandleEmailDeliveryTask).
				AddHandler(TypeImageResize, NewImageProcessor())
		}).
		Run()
}