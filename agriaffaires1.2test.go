package main

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Define the Tractor struct
type Tractor struct {
	Title              string
	Price              string
	ReferencePrice     string
	ReferenceCurrency  string
	DisplayedCurrency  string
	PriceType          string // "ex-VAT" or "inc-VAT"
	Location           string
	Dealer             string
	ImageURL           string
	DetailURL          string
	Description        string
	Details            map[string]string
}

// Define user agents for rotating requests
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
}

// Get a random user agent from the list
func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// Create an HTTP client with custom transport settings
func createClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Ignoring SSL verification
	}

	client := &http.Client{
		Timeout:   30 * time.Second, // Set a timeout for the request
		Transport: transport,
	}

	return client
}

// Make an HTTP request to a given URL
func makeRequest(url string) (*http.Response, error) {
	client := createClient()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers for the request
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Function to scrape a tractor listing page
func scrapePage(url string) {
	// Make the request to the URL
	resp, err := makeRequest(url)
	if err != nil {
		log.Fatalf("Error fetching page: %v", err)
	}
	defer resp.Body.Close()

	// Parse the response body
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Error parsing page: %v", err)
	}

	// Print the entire page HTML for debugging
	html, _ := doc.Html()
	fmt.Println(html)

	// Find the element with class js-priceToChange
	priceElement := doc.Find(".js-priceToChange")

	// Extract reference price
	referencePrice, exists := priceElement.Attr("data-reference_price")
	if !exists {
		log.Println("Reference price not found")
	}

	// Extract reference currency
	referenceCurrency, exists := priceElement.Attr("data-reference_currency")
	if !exists {
		log.Println("Reference currency not found")
	}

	// Extract displayed price (e.g., "1,736")
	displayedPrice := strings.TrimSpace(priceElement.Text())

	// Output the scraped data
	fmt.Printf("Reference Price: %s\n", referencePrice)
	fmt.Printf("Reference Currency: %s\n", referenceCurrency)
	fmt.Printf("Displayed Price: %s\n", displayedPrice)
}

// Save the list of tractors to a CSV file
func saveToCsv(tractors []Tractor) {
	err := os.MkdirAll("./results", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	filename := filepath.Join("./results", fmt.Sprintf("tractors_%s.csv", now.Format("2006-01-02_15-04-05")))

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Create CSV headers
	header := []string{"Title", "Price", "Reference Price", "Reference Currency", "Displayed Currency", "Price Type", "Location", "Dealer", "Image URL", "Detail URL", "Description"}
	if err := writer.Write(header); err != nil {
		log.Fatal(err)
	}

	// Write data rows
	for _, tractor := range tractors {
		row := []string{
			tractor.Title,
			tractor.Price,
			tractor.ReferencePrice,
			tractor.ReferenceCurrency,
			tractor.DisplayedCurrency,
			tractor.PriceType,
			tractor.Location,
			tractor.Dealer,
			tractor.ImageURL,
			tractor.DetailURL,
			tractor.Description,
		}
		if err := writer.Write(row); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("Results saved to %s\n", filename)
}

// Main function
func main() {
	baseURL := "https://www.agriaffaires.co.uk/used/farm-tractor/1/16730/fordson-major.html"
	scrapePage(baseURL)

	// Assuming you have a function to scrape multiple tractors
	var allTractors []Tractor

	// Placeholder for adding more tractor data
	// allTractors = append(allTractors, ...)

	saveToCsv(allTractors)
}
