package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

func do(duration time.Duration, eventsPerSecond int, wait time.Duration) {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	log.Printf("testing with runFor=%s, eventsPerSecond=%d, wait=%s", duration, eventsPerSecond, wait)

	ctx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	after := 10 * time.Second

	go func() {
		time.Sleep(duration + after + wait + 5*time.Second)
		cancel()
	}()

	ex := make(chan int64, 1)
	go func() {
		count := run(ctx)
		ex <- count
	}()

	time.Sleep(after)

	emitted := emit(ctx, eventsPerSecond, duration)
	executed := <-ex

	log.Printf("emitted %d, executed %d, using %d events/s", emitted, executed, eventsPerSecond)

	if emitted != executed {
		log.Fatal("emitted and executed counts do not match")
	}
}
