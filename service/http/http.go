package http

import (
	"net"
	"net/http"
	"os"
	"path"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pingcap/log"
	conprofhttp "github.com/pingcap/ng_monitoring/component/conprof/http"
	topsqlsvc "github.com/pingcap/ng_monitoring/component/topsql/service"
	"github.com/pingcap/ng_monitoring/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	httpServer *http.Server = nil
)

func ServeHTTP(l *config.Log, listener net.Listener) {
	gin.SetMode(gin.ReleaseMode)
	ng := gin.New()

	logFileName := path.Join(l.Path, "service.log")
	file, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Failed to open the log file", zap.String("filename", logFileName))
	}
	ng.Use(gin.LoggerWithWriter(file))

	// recovery
	ng.Use(gin.Recovery())

	// route
	configGroup := ng.Group("/config")
	config.HTTPService(configGroup)
	topSQLGroup := ng.Group("/topsql")
	topsqlsvc.HTTPService(topSQLGroup)
	// register pprof http api
	pprof.Register(ng)

	continuousProfilingGroup := ng.Group("/continuous_profiling")
	conprofhttp.HTTPService(continuousProfilingGroup)

	promHandler := promhttp.Handler()
	promGroup := ng.Group("/metrics")
	promGroup.Any("", func(c *gin.Context) {
		promHandler.ServeHTTP(c.Writer, c.Request)
	})

	httpServer = &http.Server{Handler: ng}
	if err = httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Warn("failed to serve http service", zap.Error(err))
	}
}

func StopHTTP() {
	if httpServer == nil {
		return
	}

	log.Info("shutting down http server")
	_ = httpServer.Close()
	log.Info("http server is down")
}
