// Package main demonstrates SSE with JSON data.
package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/sagerlabs/awesome/sse"
)

// Stock represents a stock price update
type Stock struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
	Time   string  `json:"time"`
}

// Weather represents weather data
type Weather struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
	Time        string  `json:"time"`
}

func main() {
	server := sse.NewServer()

	// Send stock updates
	go func() {
		stocks := map[string]float64{
			"AAPL": 175.50,
			"GOOGL": 140.25,
			"MSFT": 380.75,
			"TSLA": 250.00,
		}

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Pick a random stock
			symbols := make([]string, 0, len(stocks))
			for s := range stocks {
				symbols = append(symbols, s)
			}
			symbol := symbols[rand.Intn(len(symbols))]

			// Random price change
			change := (rand.Float64() - 0.5) * 4
			stocks[symbol] += change

			stock := Stock{
				Symbol: symbol,
				Price:  stocks[symbol],
				Change: change,
				Time:   time.Now().Format(time.RFC3339),
			}

			event := &sse.Event{
				Event: "stock-update",
			}
			if err := event.MarshalData(stock); err != nil {
				fmt.Printf("Error marshaling stock: %v\n", err)
				continue
			}
			server.Publish(event)
			fmt.Printf("Published stock update: %s %.2f (%.2f)\n", stock.Symbol, stock.Price, stock.Change)
		}
	}()

	// Send weather updates
	go func() {
		cities := []string{"New York", "London", "Tokyo", "Berlin"}
		conditions := []string{"Sunny", "Cloudy", "Rainy", "Partly Cloudy"}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			weather := Weather{
				City:        cities[rand.Intn(len(cities))],
				Temperature: 15 + rand.Float64()*20,
				Condition:   conditions[rand.Intn(len(conditions))],
				Humidity:    40 + rand.Intn(40),
				Time:        time.Now().Format(time.RFC3339),
			}

			event := &sse.Event{
				Event: "weather-update",
			}
			if err := event.MarshalData(weather); err != nil {
				fmt.Printf("Error marshaling weather: %v\n", err)
				continue
			}
			server.Publish(event)
			fmt.Printf("Published weather update: %s %.1f°C %s\n", weather.City, weather.Temperature, weather.Condition)
		}
	}()

	http.Handle("/events", server)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
	<style>
		.stock { padding: 10px; margin: 5px; background: #e8f5e9; border-radius: 5px; }
		.weather { padding: 10px; margin: 5px; background: #e3f2fd; border-radius: 5px; }
		.positive { color: #2e7d32; }
		.negative { color: #c62828; }
	</style>
</head>
<body>
	<h1>SSE with JSON Data</h1>
	<div id="updates"></div>
	<script>
		const eventSource = new EventSource('/events');
		const updatesDiv = document.getElementById('updates');
		
		eventSource.addEventListener('stock-update', function(event) {
			const data = JSON.parse(event.data);
			const changeClass = data.change >= 0 ? 'positive' : 'negative';
			const changeSign = data.change >= 0 ? '+' : '';
			const div = document.createElement('div');
			div.className = 'stock';
			div.innerHTML = '<strong>' + data.symbol + '</strong>: $' + data.price.toFixed(2) + 
				' <span class="' + changeClass + '">(' + changeSign + data.change.toFixed(2) + ')</span>' +
				'<br><small>' + new Date(data.time).toLocaleTimeString() + '</small>';
			updatesDiv.insertBefore(div, updatesDiv.firstChild);
		});
		
		eventSource.addEventListener('weather-update', function(event) {
			const data = JSON.parse(event.data);
			const div = document.createElement('div');
			div.className = 'weather';
			div.innerHTML = '<strong>' + data.city + '</strong>: ' + data.temperature.toFixed(1) + '°C, ' + 
				data.condition + ', ' + data.humidity + '% humidity' +
				'<br><small>' + new Date(data.time).toLocaleTimeString() + '</small>';
			updatesDiv.insertBefore(div, updatesDiv.firstChild);
		});
		
		eventSource.onerror = function(err) {
			console.error('EventSource failed:', err);
		};
	</script>
</body>
</html>
`)
	})

	fmt.Println("Server starting on :8082...")
	fmt.Println("Visit http://localhost:8082 to see the example")
	http.ListenAndServe(":8082", nil)
}
