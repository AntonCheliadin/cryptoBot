package cryptoBot

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func (s *Server) Run(port string, handler http.Handler) error {
	s.httpServer = &http.Server{
		Addr:           ":" + port,
		Handler:        handler,
		MaxHeaderBytes: 1 << 20, // 1 MB
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
	}

	if enabled, err := strconv.ParseBool(os.Getenv("HTTPS_ENABLED")); enabled && err == nil {
		return s.httpServer.ListenAndServeTLS("YOURPUBLIC.pem", "YOURPRIVATE.key")
	} else {
		return s.httpServer.ListenAndServe()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
