package main

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"log"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Tractor struct to hold the scraped data
type Tractor struct {
	URL                    string
	Details                map[string]string
	Category               string
	AdType                 string
	Reference              string
	Make                   string
	Model                  string
	Status                 string
	Power                  string
	FrontTireDimension     string
	FrontTireWear          string
	RearTireWear           string
	SparePartsAvailability string
	PriceExclVAT		   string
	DisplayedPrice         string
	ReferencePrice         string
	ReferenceCurrency      string
	PriceAdditionalInfo    string
	VATInfo                string
	Dealer                 string
	Location               string
	PhoneNumber            string
	Comments               string
	URLs				   []string
}

func createClient() *http.Client {
	// Set up an HTTP client with custom settings (TLSConfig)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   30 * time.Second, // Timeout for the request
		Transport: transport,
	}

	return client
}

func makeRequest(url string) (*http.Response, error) {
	client := createClient()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers to avoid being blocked by the website
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")

	// Set UK-specific headers
	req.Header.Set("X-Forwarded-For", "81.2.69.142") // An example UK IP address
	req.Header.Set("CF-IPCountry", "GB")

	// Set a cookie to indicate UK preference
	req.AddCookie(&http.Cookie{Name: "country_code", Value: "gb"})

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func randomDelay() {
	delay := rng.Intn(2) + 2 // Random delay between 2 to 4 seconds
	time.Sleep(time.Duration(delay) * time.Second)
}

// Scrapes the listing page to get tractor URLs and prices
func scrapeListing(url string) ([]string, error) {
	log.Printf("Sraping listing page: %s", url)
	randomDelay()
	resp, err := makeRequest(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching listing page: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing listing page: %v", err)
	}

	var urls []string
	baseURL := "https://www.agriaffaires.co.uk"

	doc.Find(".listing-block__link").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			// fullURL := baseURL + href
			urls = append(urls, baseURL+href)
		}
	})
	log.Printf("Found %d tractor URLs on listing page", len(urls))
	return urls, nil
}

// Scrapes detailed tractor info from each tractor page
func scrapePage(url string) (*Tractor, error) {
	log.Printf("Scraping tractor page:%s", url)
	resp, err := makeRequest(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching page: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing page: %v", err)
	}

	tractor := &Tractor{
		URL:     url,
		Details: make(map[string]string),
	}

	// Scrape table data
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		key := strings.TrimSpace(s.Find("td").First().Text())
		value := strings.TrimSpace(s.Find("td").Last().Text())

		// Remove trailing colon and any extra spaces from key
		key = strings.TrimSpace(strings.TrimSuffix(key, ":"))

		fmt.Printf("Scraped Key: %q, Value: %q\n", key, value)

		if key == "" || value == "" {
			return
		}

		tractor.Details[key] = value

		switch key {
		case "Category":
			tractor.Category = value
		case "Type of ad":
			tractor.AdType = value
		case "Reference":
			tractor.Reference = value
		case "Make":
			tractor.Make = value
		case "Model":
			tractor.Model = value
		case "Status":
			tractor.Status = value
		case "Power":
			tractor.Power = strings.TrimSpace(strings.Split(value, "hp")[0])
		case "Dimension of front tires":
			tractor.FrontTireDimension = value
		case "Wear of front tires":
			tractor.FrontTireWear = value
		case "Wear of rear tires":
			tractor.RearTireWear = value
		case "Period of availability of spare parts":
			tractor.SparePartsAvailability = value
		case "Comments":
			tractor.Comments = value
		}
	})

	// Handle the price information
	priceBlock := doc.Find(".block--all.block--price")

	// Extract the displayed price
	priceElement := priceBlock.Find(".js-priceToChange")
	displayedPrice := strings.TrimSpace(priceElement.Text())

	// Extract currency symbol
	currencySymbol := strings.TrimSpace(priceBlock.Find(".js-currencyToChange").Text())

	// Combine price and currency
	tractor.DisplayedPrice = displayedPrice + " " + currencySymbol

	// Extract reference price and currency
	tractor.ReferencePrice, _ = priceElement.Attr("data-reference_price")
	tractor.ReferenceCurrency, _ = priceElement.Attr("data-reference_currency")

	// Extract VAT information
	vatInfo := strings.TrimSpace(priceBlock.Find(".h3-like.u-bold").Text())
	tractor.VATInfo = vatInfo

	// Combine displayed price with VAT info
	tractor.DisplayedPrice += " " + vatInfo

	// Extract dealer information
	tractor.Dealer = strings.TrimSpace(doc.Find(".block--contact-desktop .u-bold.h3-like.man").First().Text())
	tractor.Location = strings.TrimSpace(doc.Find(".block--contact-desktop .u-bold").Last().Text())

	// Extract phone number
	phoneElement := doc.Find(".js-hi-t").First()
	tractor.PhoneNumber, _ = phoneElement.Attr("data-pdisplay")

	log.Printf("Succsessfully scraped tractor page: %s", url)
	return tractor, nil
}

// Function to save tractor data to a CSV file
func saveToCSV(tractors []*Tractor) error {
	// Get the current date and time for the filename
	now := time.Now()
	timestamp := now.Format("2006-01-02_15-04-05")

	// Create the results directory if it doesn't exist
	resultsDir := "./results"
	if err := os.MkdirAll(resultsDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating results directory: %v", err)
	}

	// Create the CSV file with the timestamp in the filename
	filename := filepath.Join(resultsDir, fmt.Sprintf("tractor_data_%s.csv", timestamp))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header
	headers := []string{
		"URLs", "URL", "Category", "Type of ad", "Reference", "Make", "Model", "Status", "Power",
		"Front Tire Dimension", "Front Tire Wear", "Rear Tire Wear", "Spare Parts Availability",
		"Price (excl. VAT)", "Displayed Price", "Reference Price", "Reference Currency",
		"VAT Info", "Dealer", "Location", "Phone Number", "Comments",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("error writing CSV header: %v", err)
	}

	// Write each tractor's details to the CSV
	for _, tractor := range tractors {
		record := []string{
			strings.Join(tractor.URLs, "|"), tractor.URL, tractor.Category, tractor.AdType, tractor.Reference, tractor.Make, tractor.Model,
			tractor.Status, tractor.Power, tractor.FrontTireDimension, tractor.FrontTireWear, tractor.RearTireWear,
			tractor.SparePartsAvailability, tractor.PriceExclVAT, tractor.DisplayedPrice, tractor.ReferencePrice, tractor.ReferenceCurrency,
			tractor.VATInfo, tractor.Dealer, tractor.Location, tractor.PhoneNumber, tractor.Comments,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error writing CSV record: %v", err)
		}
	}

	fmt.Printf("Data successfully written to %s\n", filename)
	return nil
}

func main() {
	listingURL := "https://www.agriaffaires.co.uk/used/farm-tractor/1/16730/fordson-major.html"
	tractorURLs, err := scrapeListing(listingURL)
	if err != nil {
		fmt.Printf("Error scraping listing: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	results := make(chan *Tractor, len(tractorURLs))

	for _, url := range tractorURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			tractor, err := scrapePage(url)
			if err != nil {
				fmt.Printf("Error scraping page %s: %v\n", url, err)
				return
			}
			results <- tractor
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var tractors []*Tractor
	for tractor := range results {
		tractors = append(tractors, tractor)
	}

	// Save the scraped data to CSV
	if err := saveToCSV(tractors); err != nil {
		fmt.Printf("Error saving data to CSV: %v\n", err)
	}
}