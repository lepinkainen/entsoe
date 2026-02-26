package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// PricePoint represents a single price data point with timestamp information
type PricePoint struct {
	TimeStamp      time.Time
	Time           string
	Price          float64
	RedisTimestamp int64
}

type fillOptions struct {
	Debug bool
}

// fillFromEntsoe retrieves price data from Entsoe API and stores it in Redis
func fillFromEntsoe(rdb *redis.Client, startApi, endApi string, opts fillOptions) error {

	var count int

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shouldStore := !opts.Debug

	if shouldStore {
		// ping and handle error properly
		if err := rdb.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis connection failed: %w", err)
		}

		// Create a new Redis Time Series with duplicate policy LAST
		tsOptions := redis.TSOptions{
			DuplicatePolicy: "LAST",
			Labels:          map[string]string{"type": "price", "country": "fi"},
		}

		if err := rdb.TSCreateWithArgs(ctx, viper.GetString("redis.dbname"), &tsOptions).Err(); err != nil {
			if !strings.Contains(err.Error(), "key already exists") {
				return fmt.Errorf("failed to create time series: %w", err)
			}
		}
	}

	url := fmt.Sprintf("https://web-api.tp.entsoe.eu/api?securityToken=%s&documentType=A44&out_Domain=%s&in_Domain=%s&periodStart=%s&periodEnd=%s",
		viper.GetString("nordpool.apikey"),
		viper.GetString("nordpool.in_domain"),
		viper.GetString("nordpool.out_domain"),
		startApi,
		endApi)

	//fmt.Printf("URL: %s\n", url)

	// Create HTTP client with timeouts and retry logic
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var xmlData []byte
	var err error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := client.Get(url)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("HTTP request failed after %d attempts: %w", maxRetries, err)
			}
			slog.Debug("HTTP request failed, retrying...", "attempt", attempt, "error", err)
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // exponential backoff
			continue
		}

		xmlData, err = io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Silently handle close error - not critical for functionality
			_ = closeErr
		}

		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("reading HTTP response failed after %d attempts: %w", maxRetries, err)
			}
			slog.Debug("Reading HTTP response failed, retrying...", "attempt", attempt, "error", err)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		// Retry on server errors (5xx)
		if resp.StatusCode >= 500 {
			if attempt == maxRetries {
				return fmt.Errorf("HTTP request failed after %d attempts: server returned %d", maxRetries, resp.StatusCode)
			}
			slog.Debug("Server error, retrying...", "attempt", attempt, "status", resp.StatusCode)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			continue
		}

		// Non-retryable HTTP error
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
		}

		// Success - break out of retry loop
		break
	}

	// Check that the response looks like XML before trying to parse it
	trimmed := strings.TrimSpace(string(xmlData))
	if !strings.HasPrefix(trimmed, "<?xml") && !strings.HasPrefix(trimmed, "<") {
		preview := trimmed
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return fmt.Errorf("ENTSOE API returned non-XML response: %s", preview)
	}

	var doc PublicationMarketDocument
	err = xml.Unmarshal(xmlData, &doc)
	if err != nil {
		// unmarshal error is most likely an error document
		var ackDoc AcknowledgementMarketDocument
		ackErr := xml.Unmarshal(xmlData, &ackDoc)
		if ackErr != nil {
			// Neither document type matched — log a preview of the response
			preview := trimmed
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			return fmt.Errorf("ENTSOE API returned unexpected response: %s", preview)
		}
		return fmt.Errorf("ENTSOE API error %s: %s", ackDoc.Reason.Code, ackDoc.Reason.Text)
	}

	layout := "2006-01-02T15:04Z"

	var pricePoints []PricePoint
	for _, timeserie := range doc.TimeSeries {
		startStr := timeserie.Period.TimeInterval.Start
		start, err := time.Parse(layout, startStr)
		if err != nil {
			return fmt.Errorf("failed to parse period start %q: %w", startStr, err)
		}

		resolutionDuration, err := isoResolutionToDuration(timeserie.Period.Resolution)
		if err != nil {
			if opts.Debug {
				fmt.Printf("warning: unknown resolution %q, defaulting to 1h\n", timeserie.Period.Resolution)
			}
			resolutionDuration = time.Hour
		}

		for _, point := range timeserie.Period.Points {
			pointTime := start.Add(time.Duration(point.Position-1) * resolutionDuration)

			pricePoint := PricePoint{
				TimeStamp:      pointTime,
				Time:           pointTime.Format(layout),
				Price:          point.PriceAmount / 10,
				RedisTimestamp: pointTime.Unix() * 1000,
			}

			pricePoints = append(pricePoints, pricePoint)
		}
	}

	if opts.Debug {
		fmt.Printf("Debug mode enabled; parsed %d price points across %d time series.\n", len(pricePoints), len(doc.TimeSeries))
		for _, pp := range pricePoints {
			fmt.Printf("%s -> %.2f c/kWh\n", pp.Time, pp.Price)
		}
		return nil
	}

	_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, pricePoint := range pricePoints {
			if err := pipe.TSAdd(ctx, viper.GetString("redis.dbname"),
				pricePoint.RedisTimestamp,
				pricePoint.Price).Err(); err != nil {
				return fmt.Errorf("failed to add value to Redis: %w", err)
			}
			count++
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	slog.Debug("Stored price points", "count", count)

	return nil
}

func main() {
	debugFlag := flag.Bool("debug", false, "Print parsed data for the requested window without writing to Redis")
	flag.Parse()

	// Load configuration from file
	viper.SetConfigName("config") // Name of the configuration file (without extension)
	viper.SetConfigType("yaml")   // Set the configuration type
	viper.AddConfigPath("$HOME/.config/entsoe_redis")
	viper.AddConfigPath("$HOME/.entsoe_redis")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	// Initialize Redis client using configuration from file
	rdb := redis.NewClient(&redis.Options{
		Addr:     viper.GetString("redis.address"),
		Username: viper.GetString("redis.username"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})

	now := time.Now().Truncate(time.Hour)
	if *debugFlag {
		fmt.Printf("Debug run at %s (%d)\n", now.Format(time.RFC3339), now.UnixMilli())
		fmt.Println("Debug mode enabled; data will not be stored in Redis.")
	}

	// midnight today
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format("200601020000")
	// midnight tomorrow
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Format("200601020000")

	if err := fillFromEntsoe(rdb, start, end, fillOptions{Debug: *debugFlag}); err != nil {
		slog.Error("Failed to fill data from Entsoe", "error", err)
		os.Exit(1)
	}
}

func isoResolutionToDuration(resolution string) (time.Duration, error) {
	res := strings.TrimSpace(resolution)
	if res == "" {
		return 0, fmt.Errorf("empty resolution")
	}
	if !strings.HasPrefix(res, "PT") {
		return 0, fmt.Errorf("unsupported ISO-8601 duration: %s", res)
	}
	value := strings.TrimPrefix(res, "PT")
	if strings.HasSuffix(value, "H") {
		hours, err := strconv.Atoi(strings.TrimSuffix(value, "H"))
		if err != nil {
			return 0, fmt.Errorf("invalid hour resolution %s: %w", res, err)
		}
		return time.Duration(hours) * time.Hour, nil
	}
	if strings.HasSuffix(value, "M") {
		minutes, err := strconv.Atoi(strings.TrimSuffix(value, "M"))
		if err != nil {
			return 0, fmt.Errorf("invalid minute resolution %s: %w", res, err)
		}
		return time.Duration(minutes) * time.Minute, nil
	}
	if strings.HasSuffix(value, "S") {
		seconds, err := strconv.Atoi(strings.TrimSuffix(value, "S"))
		if err != nil {
			return 0, fmt.Errorf("invalid second resolution %s: %w", res, err)
		}
		return time.Duration(seconds) * time.Second, nil
	}
	return 0, fmt.Errorf("unsupported resolution unit in %s", res)
}
