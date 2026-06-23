package main

import (
	"fmt"
	"time"
)

func waitForRegistration(registered <-chan error, timeout time.Duration) error {
	select {
	case err := <-registered:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timed out after %s waiting for workflow registration", timeout)
	}
}
