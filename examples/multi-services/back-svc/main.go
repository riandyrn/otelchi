package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const addr = ":8091"

func main() {
	// define router
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		rndNum := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(100000)
		name := fmt.Sprintf("user_%v", rndNum)
		w.Write([]byte(name))
	})
	log.Printf("back service is listening on %v", addr)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatalf("unable to execute server due: %v", err)
	}
}
