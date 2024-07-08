package pgqueue

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/lib/pq"
)

// Test_do purely tests that the postgres instance supports NOTIFY & LISTEN
func Test_do(t *testing.T) {
	do()
}

func waitForNotification(l *pq.Listener) {
	for {
		select {
		case msg := <-l.Notify:
			log.Printf("received notification: %+v", msg)
			return
		case <-time.After(90 * time.Second):
			go func() {
				_ = l.Ping()
			}()
			// Check if there's more work available, just in case it takes
			// a while for the Listener to notice connection loss and
			// reconnect.
			log.Println("received no work for 90 seconds, checking for new work")
		}
	}
}

func do() {
	var conninfo = "user=hatchet password=hatchet dbname=hatchet sslmode=disable host=localhost port=5431"

	db, err := sql.Open("postgres", conninfo)
	if err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(3 * time.Second)

		if _, err := db.Query("NOTIFY getwork, 'stuff'"); err != nil {
			panic(err)
		}
	}()

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	minReconn := 10 * time.Second
	maxReconn := time.Minute
	listener := pq.NewListener(conninfo, minReconn, maxReconn, reportProblem)
	err = listener.Listen("getwork")
	if err != nil {
		panic(err)
	}
	listener.NotificationChannel()

	fmt.Println("entering main loop")

	// process all available work before waiting for notifications
	waitForNotification(listener)
}
