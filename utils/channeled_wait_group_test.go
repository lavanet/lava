package utils

import (
	"testing"
	"time"
)

func TestChanneledWaitGroup(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		wg := NewChanneledWaitGroup()
		wg.Add(2)

		go func() {
			time.Sleep(50 * time.Millisecond)
			wg.Done()
		}()

		go func() {
			time.Sleep(100 * time.Millisecond)
			wg.Done()
		}()

		select {
		case <-wg.Wait():
			// Success
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for goroutines")
		}
	})

	t.Run("zero count should complete immediately", func(t *testing.T) {
		wg := NewChanneledWaitGroup()

		select {
		case <-wg.Wait():
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should complete immediately with zero count")
		}
	})

	t.Run("multiple waits should all receive completion", func(t *testing.T) {
		wg := NewChanneledWaitGroup()
		wg.Add(1)

		// Start three goroutines waiting
		for i := 0; i < 3; i++ {
			go func() {
				select {
				case <-wg.Wait():
					// Success
				case <-time.After(200 * time.Millisecond):
					t.Error("timeout waiting for completion")
				}
			}()
		}

		time.Sleep(50 * time.Millisecond) // Give waiters time to start
		wg.Done()
		time.Sleep(100 * time.Millisecond) // Give waiters time to complete
	})

	t.Run("reuse after completion", func(t *testing.T) {
		wg := NewChanneledWaitGroup()
		wg.Add(1)
		wg.Done()

		// First wait should complete
		select {
		case <-wg.Wait():
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("first wait should complete")
		}

		// Reset and use again
		wg.Add(1)
		go func() {
			time.Sleep(50 * time.Millisecond)
			wg.Done()
		}()

		select {
		case <-wg.Wait():
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("second wait should complete")
		}
	})
}
