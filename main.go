package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/api.git/config"
	"github.com/api.git/files"
	pb "github.com/api.git/github.com/lenush21/file_data"
	goflag "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	var flagData struct {
		Config string `short:"c" long:"config" default:"config.yaml" required:"true"`
	}

	if _, err := goflag.Parse(&flagData); err != nil {
		log.Fatal(err)
	}

	cfg, err := config.ParseConfig(flagData.Config)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Validate(); err != nil {
		log.Fatalf("error validate config: %s", err.Error())
	}

	logger := newLogger(cfg.App.LogLevel)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.HTTPPort))
	if err != nil {
		logger.WithError(err).Fatal("listener error")
	}

	var options []grpc.ServerOption

	store := files.NewDiskFileStore(&cfg)
	grpcServer := grpc.NewServer(options...)
	pb.RegisterFileDataServer(grpcServer, store)

	logger.Info("server is starting")
	if err := grpcServer.Serve(listener); err != nil {
		logger.WithError(err).Fatal("server serve error")
	}
}

func newLogger(lvl string) *logrus.Entry {
	logLevel := logrus.InfoLevel

	switch lvl {
	case "panic":
		logLevel = logrus.PanicLevel
	case "fatal":
		logLevel = logrus.FatalLevel
	case "error":
		logLevel = logrus.ErrorLevel
	case "warn":
		logLevel = logrus.WarnLevel
	case "info":
		logLevel = logrus.InfoLevel
	case "debug":
		logLevel = logrus.DebugLevel
	case "trace":
		logLevel = logrus.TraceLevel
	default:
		break
	}

	return logrus.NewEntry(&logrus.Logger{
		Formatter: &logrus.JSONFormatter{},
		Level:     logLevel,
		Out:       os.Stderr,
	})
}
