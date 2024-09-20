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

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	// Add more user agents as needed
}

func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

func createClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
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

func main() {
	baseURL := "https://www.agriaffaires.co.uk/used/farm-tractor/1/16730/fordson-major.html"
	var allTractors []Tractor

	rand.Seed(time.Now().UnixNano())

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?page=%d", baseURL, page)
		fmt.Printf("Scraping page %d: %s\n", page, url)

		pageTractors, hasNextPage, err := scrapePage(url)
		if err != nil {
			log.Printf("Error scraping page %d: %v", page, err)
			break
		}
		fmt.Printf("Found %d tractors on page %d\n", len(pageTractors), page)
		
		allTractors = append(allTractors, pageTractors...)
		fmt.Printf("Total tractors collected so far: %d\n", len(allTractors))

		if len(pageTractors) == 0 || !hasNextPage {
			fmt.Printf("No more tractors found or no next page. Stopping.\n")
			break
		}

		delay()
	}

	fmt.Printf("Total tractors found: %d\n", len(allTractors))

	for i := range allTractors {
		fmt.Printf("Scraping detailed page for tractor %d/%d\n", i+1, len(allTractors))
		scrapeDetailedPage(&allTractors[i])
		delay()
	}

	saveToCsv(allTractors)
	fmt.Printf("Results saved to results/fordson_major_tractors_%s.csv\n", time.Now().Format("2006-01-02_15-04-05"))
	fmt.Printf("Total tractors scraped: %d\n", len(allTractors))
}

const (
	baseDelay = 2 * time.Second
	jitter    = 2 * time.Second
)

func delay() {
	time.Sleep(baseDelay + time.Duration(float64(jitter)*rand.Float64()))
}

func scrapePage(url string) ([]Tractor, bool, error) {
	resp, err := makeRequest(url)
	if err != nil {
		return nil, false, fmt.Errorf("error fetching page %s: %v", url, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("error parsing page %s: %v", url, err)
	}

	
	var tractors []Tractor

	doc.Find(".listing-block.listing-block--classified").Each(func(i int, s *goquery.Selection) {
		tractor := Tractor{
			Details: make(map[string]string),
		}

		tractor.Title = strings.TrimSpace(s.Find(".listing-block__title").Text())
		tractor.Location = strings.TrimSpace(s.Find(".listing-block__localisation").Text())
		tractor.Dealer = strings.TrimSpace(s.Find(".listing-block__category").Text())
		tractor.ImageURL, _ = s.Find(".listing-block__picture img").Attr("src")
		tractor.DetailURL, _ = s.Find(".listing-block__link").Attr("href")
		if !strings.HasPrefix(tractor.DetailURL, "http") {
			tractor.DetailURL = "https://www.agriaffaires.co.uk" + tractor.DetailURL
		}

		// Enhanced price information
		
		priceElement := s.Find(".listing-block__price")
		tractor.Price = strings.TrimSpace(priceElement.Find(".js-priceToChange").Text())
		tractor.ReferencePrice, _ = priceElement.Find(".js-priceToChange").Attr("data-reference_price")
		tractor.ReferenceCurrency, _ = priceElement.Find(".js-priceToChange").Attr("data-reference_currency")
		tractor.DisplayedCurrency = strings.TrimSpace(priceElement.Find(".js-currencyToChange").Text())
		
		vatText := strings.TrimSpace(priceElement.Find(".h3-like.u-bold").Text())
		tractor.PriceType = vatText

		s.Find(".listing-block__description span").Each(func(i int, span *goquery.Selection) {
			text := strings.TrimSpace(span.Text())
			if strings.Contains(text, "hp") {
				tractor.Details["Power"] = text
			} else if strings.Contains(text, "Year") {
				tractor.Details["Year"] = text
			}
		})

		tractors = append(tractors, tractor)
		fmt.Printf("Found tractor: %s, Price: %s %s (%s)\n", tractor.Title, tractor.Price, tractor.DisplayedCurrency, tractor.PriceType)
	})

	// Check if there's a next page
	hasNextPage := false
	doc.Find(".pagination__link").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "Next") {
			hasNextPage = true
		}
	})
	fmt.Printf("Has next page: %v\n", hasNextPage)

	return tractors, hasNextPage, nil
}




func scrapeDetailedPage(tractor *Tractor) {
	resp, err := makeRequest(tractor.DetailURL)
	if err != nil {
		log.Printf("Error fetching detailed page %s: %v", tractor.DetailURL, err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Error parsing detailed page %s: %v", tractor.DetailURL, err)
		return
	}

	tractor.Description = strings.TrimSpace(doc.Find("#description_original").Text())

	doc.Find(".table--specs tr").Each(func(i int, s *goquery.Selection) {
		key := strings.TrimSpace(s.Find("td:first-child").Text())
		value := strings.TrimSpace(s.Find("td:last-child").Text())
		if key != "" && value != "" {
			tractor.Details[key] = value
		}
	})
}

func saveToCsv(tractors []Tractor) {
	err := os.MkdirAll("./results", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	filename := filepath.Join("./results", fmt.Sprintf("fordson_major_tractors_%s.csv", now.Format("2006-01-02_15-04-05")))

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Collect all possible detail keys
	detailKeys := make(map[string]bool)
	for _, tractor := range tractors {
		for k := range tractor.Details {
			detailKeys[k] = true
		}
	}

	// Create header with additional fields like ReferencePrice, ReferenceCurrency, DisplayedCurrency
	header := []string{"Title", "Price", "Reference Price", "Reference Currency", "Displayed Currency", "Price Type", "Location", "Dealer", "Image URL", "Detail URL", "Description"}
	for k := range detailKeys {
		header = append(header, k)
	}
	if err := writer.Write(header); err != nil {
		log.Fatal(err)
	}

	// Write data including the new fields
	for _, tractor := range tractors {
		row := []string{
			tractor.Title,
			tractor.Price,
			tractor.ReferencePrice,      // New field
			tractor.ReferenceCurrency,   // New field
			tractor.DisplayedCurrency,   // New field
			tractor.PriceType,           // New field (VAT information)
			tractor.Location,
			tractor.Dealer,
			tractor.ImageURL,
			tractor.DetailURL,
			tractor.Description,
		}
		for k := range detailKeys {
			if v, ok := tractor.Details[k]; ok {
				row = append(row, v)
			} else {
				row = append(row, "")
			}
		}
		if err := writer.Write(row); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("Results saved to %s\n", filename)
	fmt.Printf("Total tractors scraped: %d\n", len(tractors))
}
