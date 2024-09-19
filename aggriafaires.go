package main

import (
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
	Title       string
	Price       string
	Location    string
	Dealer      string
	ImageURL    string
	DetailURL   string
	Description string
	Details     map[string]string
}

const (
	baseDelay = 2 * time.Second
	jitter    = 2 * time.Second
)

func main() {
	baseURL := "https://www.agriaffaires.co.uk/used/farm-tractor/1/16730/fordson-major.html"
	tractors := []Tractor{}

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?page=%d", baseURL, page)
		fmt.Printf("Scraping page %d: %s\n", page, url)

		newTractors := scrapePage(url)
		if len(newTractors) == 0 {
			fmt.Printf("No more tractors found on page %d. Stopping.\n", page)
			break
		}
		tractors = append(tractors, newTractors...)

		delay()
	}

	for i := range tractors {
		fmt.Printf("Scraping detailed page for tractor %d/%d\n", i+1, len(tractors))
		scrapeDetailedPage(&tractors[i])
		delay()
	}

	saveToCsv(tractors)
}

func delay() {
	time.Sleep(baseDelay + time.Duration(float64(jitter)*rand.Float64()))
}

func scrapePage(url string) []Tractor {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching page %s: %v", url, err)
		return nil
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Error parsing page %s: %v", url, err)
		return nil
	}

	tractors := []Tractor{}

	doc.Find(".listing-block.listing-block--classified").Each(func(i int, s *goquery.Selection) {
		tractor := Tractor{
			Details: make(map[string]string),
		}

		tractor.Title = strings.TrimSpace(s.Find(".listing-block__title").Text())
		tractor.Price = strings.TrimSpace(s.Find(".listing-block__price").Text())
		tractor.Location = strings.TrimSpace(s.Find(".listing-block__localisation").Text())
		tractor.Dealer = strings.TrimSpace(s.Find(".listing-block__category").Text())
		tractor.ImageURL, _ = s.Find(".listing-block__picture img").Attr("src")
		tractor.DetailURL, _ = s.Find(".listing-block__link").Attr("href")
		if !strings.HasPrefix(tractor.DetailURL, "http") {
			tractor.DetailURL = "https://www.agriaffaires.co.uk" + tractor.DetailURL
		}

		s.Find(".listing-block__description span").Each(func(i int, span *goquery.Selection) {
			text := strings.TrimSpace(span.Text())
			if strings.Contains(text, "hp") {
				tractor.Details["Power"] = text
			} else if strings.Contains(text, "Year") {
				tractor.Details["Year"] = text
			}
		})

		tractors = append(tractors, tractor)
	})

	return tractors
}

func scrapeDetailedPage(tractor *Tractor) {
	resp, err := http.Get(tractor.DetailURL)
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

	// Create header
	header := []string{"Title", "Price", "Location", "Dealer", "Image URL", "Detail URL", "Description"}
	for k := range detailKeys {
		header = append(header, k)
	}
	if err := writer.Write(header); err != nil {
		log.Fatal(err)
	}

	// Write data
	for _, tractor := range tractors {
		row := []string{
			tractor.Title,
			tractor.Price,
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

const (
	baseDelay = 2 * time.Second
	jitter    = 2 * time.Second
)

func delay() {
	time.Sleep(baseDelay + time.Duration(float64(jitter)*rand.Float64()))
}