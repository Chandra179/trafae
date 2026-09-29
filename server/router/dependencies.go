package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type dependencies struct {
	logger           *zap.Logger
	requestLog       gin.HandlerFunc
	requestBodyLimit gin.HandlerFunc
	readiness        gin.HandlerFunc
	example          gin.HandlerFunc
	bookSearch       gin.HandlerFunc
	bookProviders    gin.HandlerFunc
}

type DependenciesConfig struct {
	Logger           *zap.Logger
	RequestLog       gin.HandlerFunc
	RequestBodyLimit gin.HandlerFunc
	Readiness        gin.HandlerFunc
	Example          gin.HandlerFunc
	BookSearch       gin.HandlerFunc
	BookProviders    gin.HandlerFunc
}

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{
		logger:           deps.Logger,
		requestLog:       deps.RequestLog,
		requestBodyLimit: deps.RequestBodyLimit,
		readiness:        deps.Readiness,
		example:          deps.Example,
		bookSearch:       deps.BookSearch,
		bookProviders:    deps.BookProviders,
	}
}
