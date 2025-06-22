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

func TestConstants(t *testing.T) {
	if DEBUG != false {
		t.Errorf("Expected DEBUG to be false, got %v", DEBUG)
	}

	if DRY_RUN != false {
		t.Errorf("Expected DRY_RUN to be false, got %v", DRY_RUN)
	}
}