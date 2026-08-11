package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	dbviewer "github.com/oisis/EldenRing-SaveForge/tools/viewer"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	address := flag.String("addr", "127.0.0.1:8787", "local address used by the viewer")
	dataDirectory := flag.String("data", "./backend/gamecatalog/data", "catalog data directory")
	flag.Parse()

	data, err := loadData(*dataDirectory)
	if err != nil {
		return err
	}
	viewer, err := dbviewer.New(data)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              *address,
		Handler:           viewer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("GameCatalog DB Viewer: http://%s", *address)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve viewer: %w", err)
	}
	return nil
}

func loadData(directory string) (loader.Data, error) {
	data, err := loader.LoadDir(directory)
	if err != nil {
		return loader.Data{}, fmt.Errorf("load catalog directory: %w", err)
	}
	return data, nil
}
