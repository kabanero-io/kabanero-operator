package utils

import (
	"fmt"
	"time"
)

// GenFunc is a generic retriable function
type GenFunc func() (bool, error)

// Retry executies the given function for the specified number of retry attempts.
func Retry(attempts int, waitTime time.Duration, gf GenFunc) error {
	for i := 0; i < attempts; i++ {
		ok, err := gf()
		if err != nil {
			return err
		}

		if ok {
			return nil
		}

		time.Sleep(waitTime)
	}

	return fmt.Errorf("Retriable function did not reach the expected outcome. Retry attempts: %v. Wait time: %v", attempts, waitTime)
}
