package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

func do(runFor time.Duration, eventsPerSecond int) {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	log.Printf("testing with runFor=%s, eventsPerSecond=%d", runFor, eventsPerSecond)

	ctx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	wait := 10 * time.Second

	go func() {
		time.Sleep(runFor + wait + 5*time.Second)
		cancel()
	}()

	ex := make(chan int, 1)
	go func() {
		count := run(ctx)
		ex <- count
	}()

	time.Sleep(wait)

	emitted := emit(ctx, eventsPerSecond, runFor)
	executed := <-ex

	log.Printf("emitted %d, executed %d", emitted, executed)

	if emitted != executed {
		log.Fatal("emitted and executed counts do not match")
	}
}
