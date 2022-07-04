package server

import (
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	readerGrpc "github.com/radyatamaa/go-cqrs-microservices/reader_service/internal/article/delivery/grpc"
	readerService "github.com/radyatamaa/go-cqrs-microservices/reader_service/proto/article_reader"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

const (
	maxConnectionIdle = 5
	gRPCTimeout       = 15
	maxConnectionAge  = 5
	gRPCTime          = 10
)

func (s *server) newReaderGrpcServer() (func() error, *grpc.Server, error) {
	l, err := net.Listen("tcp", ":"+s.cfg.GRPC.Port)
	if err != nil {
		return nil, nil, errors.Wrap(err, "net.Listen")
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: maxConnectionIdle * time.Minute,
			Timeout:           gRPCTimeout * time.Second,
			MaxConnectionAge:  maxConnectionAge * time.Minute,
			Time:              gRPCTime * time.Minute,
		}),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_recovery.UnaryServerInterceptor(),
			s.im.Logger,
		),
		),
	)

	readerGrpcService := readerGrpc.NewArticleGrpcService(s.articleUsecase, s.cfg, s.zapLog)
	readerService.RegisterReaderServiceServer(grpcServer, readerGrpcService)
	grpc_prometheus.Register(grpcServer)

	if s.cfg.GRPC.Development {
		reflection.Register(grpcServer)
	}

	go func() {
		s.zapLog.Infof("Reader gRPC server is listening on port: %s", s.cfg.GRPC.Port)
		s.zapLog.Fatal(grpcServer.Serve(l))
	}()

	return l.Close, grpcServer, nil
}
