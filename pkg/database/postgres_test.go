package database

import (
	"testing"
	"time"
)

func TestDatabaseBackOffConfiguration(t *testing.T) {
	retryBackOff := newDatabaseBackOff()

	if retryBackOff.InitialInterval != time.Second {
		t.Fatalf("InitialInterval = %s, want %s", retryBackOff.InitialInterval, time.Second)
	}
	if retryBackOff.MaxInterval != maxConnectionRetryDelay {
		t.Fatalf("MaxInterval = %s, want %s", retryBackOff.MaxInterval, maxConnectionRetryDelay)
	}
	if retryBackOff.Multiplier != 2 {
		t.Fatalf("Multiplier = %v, want 2", retryBackOff.Multiplier)
	}
	if retryBackOff.RandomizationFactor != 0.2 {
		t.Fatalf("RandomizationFactor = %v, want 0.2", retryBackOff.RandomizationFactor)
	}
}
