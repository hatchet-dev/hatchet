package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	runFor := 20 * time.Second

	ctx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	go func() {
		time.Sleep(runFor + 10*time.Second)
		cancel()
	}()

	ex := make(chan int, 1)
	go func() {
		count := run(ctx)
		ex <- count
	}()

	time.Sleep(10 * time.Second)

	emitted := emit(ctx, 1*time.Second, 50, runFor)
	executed := <-ex

	log.Printf("emitted %d, executed %d", emitted, executed)

	if emitted != executed {
		log.Fatal("emitted and executed counts do not match")
	}
}
