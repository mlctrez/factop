package service

import (
	"errors"
	"github.com/mlctrez/bind"
	"log/slog"
	"net/http"
)

var _ bind.Startup = (*WebServer)(nil)
var _ bind.Shutdown = (*WebServer)(nil)

type WebServer struct {
	slog.Logger
	server *http.Server
}

func (w *WebServer) Startup() error {
	w.server = &http.Server{Addr: ":8080", Handler: w}
	go w.listen()
	return nil
}

func (w *WebServer) listen() {
	if listenErr := w.server.ListenAndServe(); listenErr != nil {
		if !errors.Is(listenErr, http.ErrServerClosed) {
			w.Error("webserver listen error", "error", listenErr)
		}
	}
}

func (w *WebServer) Shutdown() error {
	if w.server != nil {
		return w.server.Close()
	}
	return nil
}

func (w *WebServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/" {
		writer.WriteHeader(http.StatusOK)
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("ok"))
	} else {
		writer.WriteHeader(http.StatusNotFound)
	}
}
