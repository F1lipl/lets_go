// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"contentserver/internal/config"
	"contentserver/internal/event"
	"contentserver/internal/handler"
	"contentserver/internal/respository"
	"contentserver/internal/svc"
	"contentserver/internal/taskhandler"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/content-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	if strings.TrimSpace(c.DataSource) == "" {
		log.Fatal("CONTENT_DB_DSN must be configured")
	}
	if strings.TrimSpace(c.Auth.AccessSecret) == "" {
		log.Fatal("CONTENT_ACCESS_SECRET must be configured")
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	registry, err := event.NewConsumerRegistry(map[string][]string{
		"PostPublished": {"post_card"},
	})
	if err != nil {
		log.Fatal(err)
	}
	controller := event.NewTaskController()
	if err := controller.Register("post_card", taskhandler.NewPostCardHandler(respository.NewPostCardRepository())); err != nil {
		log.Fatal(err)
	}
	pipeline, err := event.NewEventPipeline(
		respository.NewDispatchRepository(ctx.DB),
		respository.NewTaskRepository(ctx.DB),
		registry, controller, event.DefaultEventPipelineConfig(),
	)
	if err != nil {
		log.Fatal(err)
	}
	pipelineCtx, cancelPipeline := context.WithCancel(context.Background())
	defer func() {
		cancelPipeline()
		<-pipeline.Done()
	}()
	pipelineResult := make(chan error, 1)
	go func() { pipelineResult <- pipeline.Run(pipelineCtx) }()
	select {
	case <-pipeline.Ready():
	case err := <-pipelineResult:
		log.Fatalf("event pipeline stopped during startup: %v", err)
	}
	ctx.OutboxNotifier = pipeline
	// The worker may use 25 seconds to drain and 5 seconds after cancellation.
	// The process shutdown window must be longer than both phases.
	proc.SetTimeToForceQuit(40 * time.Second)
	proc.AddShutdownListener(func() {
		cancelPipeline()
		<-pipeline.Done()
	})
	go func() {
		err := <-pipelineResult
		if err != nil && (!errors.Is(err, context.Canceled) || errors.Is(err, event.ErrWorkerShutdownTimeout)) {
			log.Fatalf("event pipeline stopped unexpectedly: %v", err)
		}
	}()
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
