//go:build !ci

package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
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

func TestPingRedisUnreachableReturnsError(t *testing.T) {
	// Point at a port that refuses connections so the dial fails, mimicking the
	// DNS/dial timeouts seen on the cron host. After exhausting its retries
	// pingRedis must surface the error so main() can log it and the operator can
	// intervene.
	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
		MaxRetries:  -1,
	})
	defer func() { _ = rdb.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := pingRedis(ctx, rdb)
	if err == nil {
		t.Fatal("expected an error pinging an unreachable Redis")
	}
	if !strings.Contains(err.Error(), "redis connection failed") {
		t.Fatalf("expected a redis connection failure, got %v", err)
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
