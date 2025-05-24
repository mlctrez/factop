package service

import (
	"github.com/mlctrez/servicego"
	"net/http"
)

var _ Component = (*WebServer)(nil)

type WebServer struct {
	servicego.DefaultLogger
	server *http.Server
}

func (w *WebServer) Start(s *Service) error {
	w.Logger(s.Log())
	w.server = &http.Server{Addr: ":8080", Handler: w}
	go func() {
		if listenErr := w.server.ListenAndServe(); listenErr != nil {
			w.Errorf("webserver listen error: %s", listenErr)
		}
	}()
	return nil
}

func (w *WebServer) Stop() error {
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
