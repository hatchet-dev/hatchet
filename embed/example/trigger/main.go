package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"

	_ "github.com/hatchet-dev/hatchet/embed"
)

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Worker string `json:"worker"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	databaseURL := env("DATABASE_URL")
	runs := atoiDefault(os.Getenv("RUNS"), 30)

	embedOpts := []hatchet.EmbeddedOption{hatchet.WithoutEmbeddedAPI()}
	if port := os.Getenv("GRPC_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		embedOpts = append(embedOpts, hatchet.WithEmbeddedGRPCPort(p))
	}

	client, err := hatchet.NewClient(hatchet.WithEmbeddedPostgres(databaseURL, embedOpts...))
	if err != nil {
		return err
	}
	defer func() { _ = client.Close(context.Background()) }()

	log.Printf("triggering %d runs of \"greet\"", runs) // nolint:gosec

	refs := make([]*hatchet.WorkflowRunRef, 0, runs)
	for i := 0; i < runs; i++ {
		ref, err := triggerWithRetry(ctx, client, i)
		if err != nil {
			return fmt.Errorf("trigger %d failed: %w", i, err)
		}
		refs = append(refs, ref)
	}

	counts := map[string]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, ref := range refs {
		wg.Add(1)
		go func(ref *hatchet.WorkflowRunRef) {
			defer wg.Done()
			res, err := ref.Result()
			if err != nil {
				return
			}
			var out GreetOutput
			if err := res.TaskOutput("greet").Into(&out); err != nil || out.Worker == "" {
				return
			}
			mu.Lock()
			counts[out.Worker]++
			mu.Unlock()
		}(ref)
	}
	wg.Wait()

	fmt.Println()
	fmt.Printf("distribution of %d runs across workers:\n", runs)
	names := make([]string, 0, len(counts))
	total := 0
	for w, c := range counts {
		names = append(names, w)
		total += c
	}
	sort.Strings(names)
	for _, w := range names {
		fmt.Printf("  %-12s %d\n", w, counts[w])
	}
	fmt.Printf("  %-12s %d/%d completed across %d workers\n", "total:", total, runs, len(counts))
	return nil
}

func triggerWithRetry(ctx context.Context, client *hatchet.Client, i int) (*hatchet.WorkflowRunRef, error) {
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		ref, err := client.RunNoWait(ctx, "greet", GreetInput{Name: fmt.Sprintf("run-%d", i)})
		if err == nil {
			return ref, nil
		}
		lastErr = err
		time.Sleep(time.Second)
	}
	return nil, lastErr
}

func env(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is not set", key)
	}
	return v
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
