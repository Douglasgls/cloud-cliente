package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	listener   net.Listener
}

func NewServer(listener net.Listener, handler http.Handler) *Server {
	return &Server{
		listener: listener,
		httpServer: &http.Server{
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	if s.httpServer == nil || s.listener == nil {
		return errors.New("server or listener not initialized")
	}

	err := s.httpServer.Serve(s.listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
