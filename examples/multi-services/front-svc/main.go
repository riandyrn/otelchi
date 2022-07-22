package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"golang.org/x/net/context/ctxhttp"
)

const (
	envKeyBackServiceURL = "BACK_SERVICE_URL"
	addr                 = ":8090"
)

func main() {
	// define router
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		name, err := getRandomName(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(fmt.Sprintf("Hello, %s!", name)))
	})
	log.Printf("front service is listening on %v", addr)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatalf("unable to execute server due: %v", err)
	}
}

func getRandomName(ctx context.Context) (string, error) {
	resp, err := ctxhttp.Get(ctx, http.DefaultClient, os.Getenv(envKeyBackServiceURL))
	if err != nil {
		return "", fmt.Errorf("unable to execute http request due: %w", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response data due: %w", err)
	}

	return string(data), nil
}
