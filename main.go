package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
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

const DEBUG = false
const DRY_RUN = false

// fillFromEntsoe retrieves price data from Entsoe API and stores it in Redis
func fillFromEntsoe(rdb *redis.Client, startApi, endApi string) error {

	var count = 0

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	url := fmt.Sprintf("https://web-api.tp.entsoe.eu/api?securityToken=%s&documentType=A44&out_Domain=%s&in_Domain=%s&periodStart=%s&periodEnd=%s",
		viper.GetString("nordpool.apikey"),
		viper.GetString("nordpool.in_domain"),
		viper.GetString("nordpool.out_domain"),
		startApi,
		endApi)

	//fmt.Printf("URL: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v", err)
		return err
	}
	defer resp.Body.Close()

	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading HTTP response: %v", err)
		return err
	}

	var doc PublicationMarketDocument
	err = xml.Unmarshal(xmlData, &doc)
	if err != nil {
		// unmarshal error is most likely an error document
		var doc AcknowledgementMarketDocument
		err = xml.Unmarshal(xmlData, &doc)
		if err != nil {
			fmt.Printf("Error unmarshaling AcknowledgementMarketDocumentXML: %v", err)
			return err
		}
		fmt.Printf("Error retrieving data: Code: %s\nMessage: %s\n", doc.Reason.Code, doc.Reason.Text)
		return err
	}

	layout := "2006-01-02T15:04Z"

	// Use pipeline to fill the data faster
	_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, timeserie := range doc.TimeSeries {
			// start of this timeseries
			startStr := timeserie.Period.TimeInterval.Start
			start, _ := time.Parse(layout, startStr)

			for _, point := range timeserie.Period.Points {
				// the actual time of the point
				pointTime := start.Add(time.Duration(point.Position-1) * time.Hour)

				// timestamp and actual c/kWh price (VAT included)
				pricePoint := PricePoint{
					TimeStamp:      pointTime,
					Time:           pointTime.Format(layout),
					Price:          point.PriceAmount / 10,
					RedisTimestamp: pointTime.Unix() * 1000,
				}

				if !DRY_RUN {
					if err := pipe.TSAdd(ctx, viper.GetString("redis.dbname"),
						pricePoint.RedisTimestamp,
						pricePoint.Price).Err(); err != nil {
						return fmt.Errorf("failed to add value to Redis: %w", err)
					}
					count++
				}
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	return nil
}

func main() {
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
	if DEBUG {
		fmt.Printf("%+v\n", now)
		fmt.Printf("%+v\n", now.UnixMilli())
	}

	// midnight today
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format("200601020000")
	// midnight tomorrow
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()).Format("200601020000")

	if err := fillFromEntsoe(rdb, start, end); err != nil {
		log.Fatalf("Failed to fill data from Entsoe: %v", err)
	}
}
