package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sanketlengare/orchard/internal/controlplane"
	"github.com/sanketlengare/orchard/internal/httpapi"
	"github.com/sanketlengare/orchard/internal/store"
)

const defaultListenAddress = "127.0.0.1:8080"

func newHTTPHandler() (http.Handler, error) {
	jobStore := store.NewMemoryJobStore()
	workerStore := store.NewMemoryWorkerStore()

	service, err := controlplane.NewService(
		jobStore,
		workerStore,
	)
	if err != nil {
		return nil, err
	}

	return httpapi.NewHandler(service), nil
}

func serveHTTP(stdout io.Writer) error {
	handler, err := newHTTPHandler()
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              defaultListenAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Fprintf(
		stdout,
		"orchard server listening on http://%s\n",
		defaultListenAddress,
	)

	return server.ListenAndServe()
}
