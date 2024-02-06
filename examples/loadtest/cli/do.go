package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

func do(runFor time.Duration, sleep time.Duration, delay time.Duration, amount int) {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	log.Printf("testing with runFor=%s, sleep=%s, delay=%s, amount=%d", runFor, sleep, delay, amount)

	ctx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	go func() {
		time.Sleep(runFor + 5*time.Second)
		cancel()
	}()

	ex := make(chan int, 1)
	go func() {
		count := run(ctx)
		ex <- count
	}()

	time.Sleep(10 * time.Second)

	emitted := emit(ctx, sleep, delay, amount, runFor)
	executed := <-ex

	log.Printf("emitted %d, executed %d", emitted, executed)

	if emitted != executed {
		log.Fatal("emitted and executed counts do not match")
	}
}
