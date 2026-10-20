package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	apiListenAddr = "0.0.0.0:20080"
	apiClientURL  = "http://127.0.0.1:20080"
)

type apiServer struct {
	server          *http.Server
	listener        net.Listener
	serveDone       chan struct{}
	serveErr        error
	errCh           chan error
	shutdownTimeout time.Duration
	stopOnce        sync.Once
	stopDone        chan struct{}
}

func startAPIServer(addr string, handler http.Handler) (*apiServer, error) {
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}
	s := &apiServer{
		server: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		listener:        ln,
		serveDone:       make(chan struct{}),
		errCh:           make(chan error, 1),
		shutdownTimeout: 5 * time.Second,
		stopDone:        make(chan struct{}),
	}
	go s.serve()
	return s, nil
}

func (s *apiServer) serve() {
	defer close(s.serveDone)
	err := s.server.Serve(s.listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.serveErr = err
		s.errCh <- err
	}
}

func (s *apiServer) Err() <-chan error { return s.errCh }

func (s *apiServer) Addr() string { return s.listener.Addr().String() }

func (s *apiServer) ServeErr() error {
	<-s.serveDone
	return s.serveErr
}

func (s *apiServer) Stop() {
	s.stopOnce.Do(func() {
		go s.shutdown()
	})
	<-s.stopDone
}

func (s *apiServer) shutdown() {
	defer close(s.stopDone)
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("API server shutdown: %v; forcing close", err)
		_ = s.server.Close()
	}
	<-s.serveDone
}
