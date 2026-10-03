package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type dependencies struct {
	logger           *zap.Logger
	requestLog       gin.HandlerFunc
	requestBodyLimit gin.HandlerFunc
	rateLimit        gin.HandlerFunc
	readiness        gin.HandlerFunc
	example          gin.HandlerFunc
	bookSearch       gin.HandlerFunc
	bookProviders    gin.HandlerFunc
	bookEvent        gin.HandlerFunc
	metrics          gin.HandlerFunc
}

type DependenciesConfig struct {
	Logger           *zap.Logger
	RequestLog       gin.HandlerFunc
	RequestBodyLimit gin.HandlerFunc
	// RateLimit throttles the API routes per client; nil disables it.
	RateLimit     gin.HandlerFunc
	Readiness     gin.HandlerFunc
	Example       gin.HandlerFunc
	BookSearch    gin.HandlerFunc
	BookProviders gin.HandlerFunc
	BookEvent     gin.HandlerFunc
	Metrics       gin.HandlerFunc
}

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{
		logger:           deps.Logger,
		requestLog:       deps.RequestLog,
		requestBodyLimit: deps.RequestBodyLimit,
		rateLimit:        deps.RateLimit,
		readiness:        deps.Readiness,
		example:          deps.Example,
		bookSearch:       deps.BookSearch,
		bookProviders:    deps.BookProviders,
		bookEvent:        deps.BookEvent,
		metrics:          deps.Metrics,
	}
}
