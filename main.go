package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type PricePoint struct {
	TimeStamp      time.Time
	Time           string
	Price          float64
	RedisTimestamp int64
}

var ctx = context.Background()

const DEBUG = false
const DRY_RUN = false

func fill_from_entsoe(rdb *redis.Client, start, end string) {

	var count = 0

	// ping and panic if failed
	res := rdb.Ping(ctx)
	if res.Err() != nil {
		panic(res.Err())
	}

	// Create a new Redis Time Series with duplicate policy LAST, allowing overwrites
	var tsOptions = redis.TSOptions{
		DuplicatePolicy: "LAST",
	}

	createRes := rdb.TSCreateWithArgs(ctx, viper.GetString("redis.dbname"), &tsOptions)

	if createRes.Err() != nil && (createRes.Err().Error() == "TS.CREATE entsoe:fi DUPLICATE_POLICY LAST: ERR TSDB: key already exists") {
		fmt.Printf("createRes: %+v\n", createRes)
	}

	url := fmt.Sprintf("https://web-api.tp.entsoe.eu/api?securityToken=%s&documentType=A44&out_Domain=%s&in_Domain=%s&periodStart=%s&periodEnd=%s",
		viper.GetString("nordpool.apikey"),
		viper.GetString("nordpool.in_domain"),
		viper.GetString("nordpool.out_domain"),
		start,
		end)

	//fmt.Printf("URL: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v", err)
		return
	}
	defer resp.Body.Close()

	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading HTTP response: %v", err)
		return
	}

	var doc PublicationMarketDocument
	err = xml.Unmarshal(xmlData, &doc)
	if err != nil {
		// unmarshal error is most likely an error document
		var doc AcknowledgementMarketDocument
		err = xml.Unmarshal(xmlData, &doc)
		if err != nil {
			fmt.Printf("Error unmarshaling AcknowledgementMarketDocumentXML: %v", err)
			return
		}
		fmt.Printf("Error retrieving data: Code: %s\nMessage: %s\n", doc.Reason.Code, doc.Reason.Text)
		return
	}

	layout := "2006-01-02T15:04Z"

	// Use pipeline to fill the data faster
	rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {

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

				tsOptions = redis.TSOptions{
					Labels: map[string]string{
						"source": "entsoe",
					},
				}

				if !DRY_RUN {
					//res := pipe.TSAdd(ctx, viper.GetString("redis.dbname"), pricePoint.RedisTimestamp, pricePoint.Price)
					res := pipe.TSAddWithArgs(ctx, viper.GetString("redis.dbname"), pricePoint.RedisTimestamp, pricePoint.Price, &tsOptions)
					if res.Err() != nil {
						fmt.Printf("Error adding value to Redis: %+v\n", res.Err())
					}

					count = count + 1

					if DEBUG {
						fmt.Printf("Result: %+v\n", res)
						fmt.Printf("PricePoint: %+v\n", pricePoint)
					}
				}
			}
		}
		return nil
	})

	//fmt.Printf("Inserted %+v values to redisdb %s\n", count, viper.GetString("redis.dbname"))
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

	fill_from_entsoe(rdb, start, end)
}

/*
	fmt.Println()

	// get current price
	nowUnix := int(now.UnixMilli())
	valueSlice := rdb.TSRange(ctx, DBNAME, nowUnix, nowUnix)
	if valueSlice.Err() != nil {
		fmt.Printf("Error getting values from Redis: %+v\n", valueSlice.Err())
	}
	tsValue, _ := valueSlice.Result()
	fmt.Printf("Redis values: %+v\n", tsValue)
	if len(tsValue) > 0 {
		fmt.Printf("Value: %.2f\n", tsValue[0].Value)
	}
*/
