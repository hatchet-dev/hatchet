package main

import (
	"context"
	"fmt"
	"sync"

	v1_workflows "github.com/hatchet-dev/hatchet/examples/go/workflows"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/joho/godotenv"
)

func simple() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// > Running a Task
	simple := v1_workflows.Simple(hatchet)
	result, err := simple.Run(ctx, v1_workflows.SimpleInput{
		Message: "Hello, World!",
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(result.TransformedMessage)
	

	// > Running Multiple Tasks
	var results []string
	var resultsMutex sync.Mutex
	var errs []error
	var errsMutex sync.Mutex

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		result, err := simple.Run(ctx, v1_workflows.SimpleInput{
			Message: "Hello, World!",
		})

		if err != nil {
			errsMutex.Lock()
			errs = append(errs, err)
			errsMutex.Unlock()
			return
		}

		resultsMutex.Lock()
		results = append(results, result.TransformedMessage)
		resultsMutex.Unlock()
	}()

	go func() {
		defer wg.Done()
		result, err := simple.Run(ctx, v1_workflows.SimpleInput{
			Message: "Hello, Moon!",
		})

		if err != nil {
			errsMutex.Lock()
			errs = append(errs, err)
			errsMutex.Unlock()
			return
		}

		resultsMutex.Lock()
		results = append(results, result.TransformedMessage)
		resultsMutex.Unlock()
	}()

	wg.Wait()
	

	// > Running a Task Without Waiting
	simple = v1_workflows.Simple(hatchet)
	runRef, err := simple.RunNoWait(ctx, v1_workflows.SimpleInput{
		Message: "Hello, World!",
	})

	if err != nil {
		panic(err)
	}

	// The Run Ref Exposes an ID that can be used to wait for the task to complete
	// or check on the status of the task
	runId := runRef.RunId()
	fmt.Println(runId)
	

	// > Subscribing to results
	// finally, we can wait for the task to complete and get the result
	finalResult, err := runRef.Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(finalResult)
	
}
