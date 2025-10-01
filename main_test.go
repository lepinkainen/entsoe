//go:build !ci

package main

import (
	"testing"
	"time"
)

func TestPricePoint(t *testing.T) {
	now := time.Now()
	pp := PricePoint{
		TimeStamp:      now,
		Time:           now.Format("2006-01-02T15:04Z"),
		Price:          25.5,
		RedisTimestamp: now.Unix() * 1000,
	}

	if pp.Price != 25.5 {
		t.Errorf("Expected price 25.5, got %f", pp.Price)
	}

	if pp.RedisTimestamp != now.Unix()*1000 {
		t.Errorf("Expected timestamp %d, got %d", now.Unix()*1000, pp.RedisTimestamp)
	}
}

func TestIsoResolutionToDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		expect  time.Duration
		wantErr bool
	}{
		{name: "hourly", input: "PT1H", expect: time.Hour},
		{name: "fifteen minutes", input: "PT15M", expect: 15 * time.Minute},
		{name: "seconds", input: "PT30S", expect: 30 * time.Second},
		{name: "invalid", input: "15M", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dur, err := isoResolutionToDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}

			if dur != tt.expect {
				t.Fatalf("expected %v, got %v", tt.expect, dur)
			}
		})
	}
}
