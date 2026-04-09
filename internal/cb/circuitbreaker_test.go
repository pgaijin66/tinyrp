package cb

import (
	"testing"
	"time"
)

func TestClosedToOpen(t *testing.T) {
	cb := New(3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Fatalf("should allow request %d", i)
		}
		cb.RecordFailure()
	}

	if cb.State() != Open {
		t.Fatal("expected Open after 3 failures")
	}
	if cb.Allow() {
		t.Fatal("should reject when Open")
	}
}

func TestOpenToHalfOpen(t *testing.T) {
	cb := New(1, 50*time.Millisecond)
	cb.RecordFailure()

	if cb.State() != Open {
		t.Fatal("expected Open")
	}

	time.Sleep(60 * time.Millisecond)

	if !cb.Allow() {
		t.Fatal("should allow after reset timeout (half-open)")
	}
	if cb.State() != HalfOpen {
		t.Fatal("expected HalfOpen")
	}
}

func TestHalfOpenRecovery(t *testing.T) {
	cb := New(1, 50*time.Millisecond)
	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	cb.Allow() // transitions to half-open

	cb.RecordSuccess()
	if cb.State() != Closed {
		t.Fatal("expected Closed after success in HalfOpen")
	}
}

func TestSuccessResetFailures(t *testing.T) {
	cb := New(3, 100*time.Millisecond)
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()

	// should be back to 0 failures, so one more failure should not open
	cb.RecordFailure()
	if cb.State() != Closed {
		t.Fatal("expected Closed, success should have reset counter")
	}
}
