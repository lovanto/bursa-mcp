package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type TradingResponse struct {
	KodeEmiten string        `json:"KodeEmiten"`
	Replies    []TradingData `json:"replies"`
}

type TradingData struct {
	Date        string  `json:"Date"`
	Close       float64 `json:"Close"`
	Volume      float64 `json:"Volume"`
	ForeignBuy  float64 `json:"ForeignBuy"`
	ForeignSell float64 `json:"ForeignSell"`
}

func main() {
	fmt.Println("Starting Task 1: Probe access & Cloudflare")

	url := "https://idx.co.id/primary/ListedCompany/GetTradingInfoSS?code=BBCA&length=30"

	// Using Firefox_120 as it successfully bypasses Cloudflare
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(15),
		tls_client.WithClientProfile(profiles.Firefox_120),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		fmt.Printf("Error creating tls client: %v\n", err)
		os.Exit(1)
	}

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", "https://idx.co.id/")
	req.Header.Set("Origin", "https://idx.co.id")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0")

	fmt.Printf("Fetching: %s\n", url)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Non-200 Status Code: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	body, _ := io.ReadAll(resp.Body)
	var data TradingResponse
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Success fetching data for %s. Total records: %d\n", data.KodeEmiten, len(data.Replies))

	// Print last 5 records (or first 5 if it's descending)
	fmt.Println("Printing top 5 records:")
	fmt.Printf("%-20s | %-10s | %-15s | %-15s | %-15s\n", "Date", "Close", "Volume", "ForeignBuy", "ForeignSell")
	fmt.Println("--------------------------------------------------------------------------------------")

	count := 0
	for _, row := range data.Replies {
		fmt.Printf("%-20s | %-10.0f | %-15.0f | %-15.0f | %-15.0f\n",
			row.Date, row.Close, row.Volume, row.ForeignBuy, row.ForeignSell)
		count++
		if count >= 5 {
			break
		}
	}
}
