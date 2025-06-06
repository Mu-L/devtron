/*
 * Copyright (c) 2020-2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/devtron-labs/common-lib/middlewares"
	pubsub "github.com/devtron-labs/common-lib/pubsub-lib"
	posthogTelemetry "github.com/devtron-labs/common-lib/telemetry"
	"github.com/devtron-labs/devtron/pkg/eventProcessor"
	"github.com/devtron-labs/devtron/pkg/eventProcessor/in"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/devtron-labs/devtron/api/util"
	"github.com/devtron-labs/devtron/otel"
	"github.com/devtron-labs/devtron/pkg/auth/user"

	"github.com/casbin/casbin"
	casbinv2 "github.com/casbin/casbin/v2"
	authMiddleware "github.com/devtron-labs/authenticator/middleware"
	"github.com/devtron-labs/devtron/api/router"
	"github.com/devtron-labs/devtron/api/sse"
	"github.com/devtron-labs/devtron/internal/middleware"
	"github.com/go-pg/pg"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.uber.org/zap"
)

type App struct {
	MuxRouter     *router.MuxRouter
	Logger        *zap.SugaredLogger
	SSE           *sse.SSE
	Enforcer      *casbin.SyncedEnforcer
	EnforcerV2    *casbinv2.SyncedEnforcer
	server        *http.Server
	db            *pg.DB
	posthogClient *posthogTelemetry.PosthogClient
	userService   user.UserService
	// eventProcessor.CentralEventProcessor is used to register event processors
	centralEventProcessor *eventProcessor.CentralEventProcessor // do not remove this.
	// used for local dev only
	serveTls                   bool
	sessionManager2            *authMiddleware.SessionManager
	OtelTracingService         *otel.OtelTracingServiceImpl
	loggingMiddleware          util.LoggingMiddleware
	pubSubClient               *pubsub.PubSubClientServiceImpl
	workflowEventProcessorImpl *in.WorkflowEventProcessorImpl
}

func NewApp(router *router.MuxRouter,
	Logger *zap.SugaredLogger,
	sse *sse.SSE,
	enforcer *casbin.SyncedEnforcer,
	db *pg.DB,
	sessionManager2 *authMiddleware.SessionManager,
	posthogClient *posthogTelemetry.PosthogClient,
	loggingMiddleware util.LoggingMiddleware,
	centralEventProcessor *eventProcessor.CentralEventProcessor,
	pubSubClient *pubsub.PubSubClientServiceImpl,
	workflowEventProcessorImpl *in.WorkflowEventProcessorImpl,
	enforcerV2 *casbinv2.SyncedEnforcer,
	userService user.UserService,
) *App {
	//check argo connection
	//todo - check argo-cd version on acd integration installation
	app := &App{
		MuxRouter:                  router,
		Logger:                     Logger,
		SSE:                        sse,
		Enforcer:                   enforcer,
		EnforcerV2:                 enforcerV2,
		db:                         db,
		serveTls:                   false,
		sessionManager2:            sessionManager2,
		posthogClient:              posthogClient,
		OtelTracingService:         otel.NewOtelTracingServiceImpl(Logger),
		loggingMiddleware:          loggingMiddleware,
		centralEventProcessor:      centralEventProcessor,
		pubSubClient:               pubSubClient,
		workflowEventProcessorImpl: workflowEventProcessorImpl,
		userService:                userService,
	}
	return app
}

func (app *App) Start() {
	port := 8080 //TODO: extract from environment variable
	app.Logger.Debugw("starting server")
	app.Logger.Infow("starting server on ", "port", port)

	// setup tracer
	tracerProvider := app.OtelTracingService.Init(otel.OTEL_ORCHESTRASTOR_SERVICE_NAME)

	app.MuxRouter.Init()
	//authEnforcer := casbin2.Create()

	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: authMiddleware.Authorizer(app.sessionManager2, user.WhitelistChecker, app.userService.CheckUserStatusAndUpdateLoginAudit)(app.MuxRouter.Router)}
	app.MuxRouter.Router.Use(app.loggingMiddleware.LoggingMiddleware)
	app.MuxRouter.Router.Use(middleware.PrometheusMiddleware)
	app.MuxRouter.Router.Use(middlewares.Recovery)

	if tracerProvider != nil {
		app.MuxRouter.Router.Use(otelmux.Middleware(otel.OTEL_ORCHESTRASTOR_SERVICE_NAME))
	}
	app.server = server
	var err error
	if app.serveTls {
		cert, err := tls.LoadX509KeyPair(
			"localhost.crt",
			"localhost.key",
		)
		if err != nil {
			log.Fatal(err)
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		err = server.ListenAndServeTLS("", "")
	} else {
		err = server.ListenAndServe()
	}
	//err := http.ListenAndServe(fmt.Sprintf(":%d", port), auth.Authorizer(app.Enforcer, app.sessionManager)(app.MuxRouter.Router))
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Logger.Errorw("error in startup", "err", err)
		os.Exit(2)
	}
}

func (app *App) Stop() {
	app.Logger.Info("orchestrator shutdown initiating")
	err := app.pubSubClient.ShutDown()
	if err != nil {
		app.Logger.Errorw("error in NATS client shutdown", "err", err)
	}
	app.workflowEventProcessorImpl.ShutDownDevtronAppReleaseContext()
	posthogCl := app.posthogClient.Client
	if posthogCl != nil {
		app.Logger.Info("flushing messages of posthog")
		posthogCl.Close()
	}
	timeoutContext, _ := context.WithTimeout(context.Background(), 5*time.Second)
	app.Logger.Infow("closing router")
	err = app.server.Shutdown(timeoutContext)
	if err != nil {
		app.Logger.Errorw("error in mux router shutdown", "err", err)
	}

	app.OtelTracingService.Shutdown()

	app.Logger.Infow("closing db connection")
	err = app.db.Close()
	if err != nil {
		app.Logger.Errorw("error in closing db connection", "err", err)
	}
	//Close not needed if you Drain.

	if err != nil {
		app.Logger.Errorw("Error in draining nats connection", "error", err)
	}

	app.Logger.Infow("housekeeping done. exiting now")
}
