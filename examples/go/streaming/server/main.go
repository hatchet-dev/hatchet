package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/examples/go/streaming/shared"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
)

func main() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	streamingWorkflow := shared.StreamingWorkflow(hatchet)

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		workflowRun, err := streamingWorkflow.RunNoWait(ctx, shared.StreamTaskInput{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stream, err := hatchet.Runs().SubscribeToStream(ctx, workflowRun.RunId())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		flusher, _ := w.(http.Flusher)
		for content := range stream {
			fmt.Fprint(w, content)
			if flusher != nil {
				flusher.Flush()
			}
		}
	})

	server := &http.Server{
		Addr:         ":8000",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Println("Failed to start server:", err)
	}
}
