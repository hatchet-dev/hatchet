package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/examples/go/streaming/shared"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// > Server
func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	streamingWorkflow := shared.StreamingWorkflow(client)

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

		stream := client.Runs().SubscribeToStream(ctx, workflowRun.RunId)

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
