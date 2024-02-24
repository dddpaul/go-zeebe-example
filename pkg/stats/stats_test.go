package stats

import (
	"sync"
	"testing"
)

func TestConcurrentIncrements(t *testing.T) {
	// Reset global counters before test
	created.Store(0)
	approved.Store(0)
	rejected.Store(0)

	// Define the number of increments per counter
	increments := 1000

	// Use WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(3 * increments) // 3 counters, incremented 'increments' times each

	// Increment 'created' counter in its own goroutine
	for i := 0; i < increments; i++ {
		go func() {
			IncrementCreated()
			wg.Done()
		}()
	}

	// Increment 'approved' counter in its own goroutine
	for i := 0; i < increments; i++ {
		go func() {
			IncrementApproved()
			wg.Done()
		}()
	}

	// Increment 'rejected' counter in its own goroutine
	for i := 0; i < increments; i++ {
		go func() {
			IncrementRejected()
			wg.Done()
		}()
	}

	// Wait for all increments to complete
	wg.Wait()

	// Get the current stats
	currentStats := Get()

	// Check that each counter has been incremented the correct number of times
	if currentStats.Created != int64(increments) {
		t.Errorf("Created counter incorrect: got %v, want %v", currentStats.Created, increments)
	}
	if currentStats.Approved != int64(increments) {
		t.Errorf("Approved counter incorrect: got %v, want %v", currentStats.Approved, increments)
	}
	if currentStats.Rejected != int64(increments) {
		t.Errorf("Rejected counter incorrect: got %v, want %v", currentStats.Rejected, increments)
	}
}
