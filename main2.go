package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Tractor struct {
	Title         string
	Price         string
	OriginalPrice string
	PriceExclVAT  string
	HP            string
	Year          string
	WorkingHours  string
	Dealer        string
	Location      string
	ImageURL      string
	Details       string
}

func main() {
	baseURL := "https://www.landwirt.com/en/used-farm-machinery/tractors.html"
	tractors := []Tractor{}

	// Scrape multiple pages
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?offset=%d", baseURL, (page-1)*20)
		fmt.Printf("Scraping page %d: %s\n", page, url)

		newTractors := scrapePage(url)
		if len(newTractors) == 0 {
			fmt.Printf("No more tractors found on page %d. Stopping.\n", page)
			break
		}
		tractors = append(tractors, newTractors...)
	}

	// Save results to CSV
	saveToCsv(tractors)
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

	doc.Find(".row.gmmtreffer").Each(func(i int, s *goquery.Selection) {
		tractor := Tractor{}

		tractor.Title = strings.TrimSpace(s.Find("h3 a").Text())
		tractor.Price = strings.TrimSpace(s.Find(".gmmprice1, .pricetagbig").First().Text())
		tractor.OriginalPrice = strings.TrimSpace(s.Find(".gmmprice4 s").Text())
		tractor.PriceExclVAT = strings.TrimSpace(s.Find(".gmmVat.hidden-xs").Last().Text())
		tractor.ImageURL, _ = s.Find(".bildboxgmm img").Attr("src")
		tractor.Details = strings.TrimSpace(s.Find("p[style='font-size:14px']").Text())

		s.Find(".gmmlistcatfield li").Each(func(i int, li *goquery.Selection) {
			text := strings.TrimSpace(li.Text())
			if strings.HasPrefix(text, "hp/kW:") {
				tractor.HP = strings.TrimPrefix(text, "hp/kW:")
			} else if strings.HasPrefix(text, "Year of construction:") {
				tractor.Year = strings.TrimPrefix(text, "Year of construction:")
			} else if strings.HasPrefix(text, "Working hours:") {
				tractor.WorkingHours = strings.TrimPrefix(text, "Working hours:")
			}
		})

		dealerInfo := s.Find("address.gmmlist_t10").Text()
		parts := strings.Split(dealerInfo, "-")
		if len(parts) == 2 {
			tractor.Dealer = strings.TrimSpace(parts[0])
			tractor.Location = strings.TrimSpace(parts[1])
		}

		tractors = append(tractors, tractor)
	})

	return tractors
}

func saveToCsv(tractors []Tractor) {
	// Create results directory if it doesn't exist
	err := os.MkdirAll("./results", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	// Create filename with current date and time
	now := time.Now()
	filename := filepath.Join("./results", fmt.Sprintf("tractor_results_%s.csv", now.Format("2006-01-02_15-04-05")))

	// Create and open the file
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Title", "Price", "Original Price", "Price Excl. VAT", "HP", "Year", "Working Hours", "Dealer", "Location", "Image URL", "Details"}
	if err := writer.Write(header); err != nil {
		log.Fatal(err)
	}

	// Write data
	for _, tractor := range tractors {
		row := []string{
			tractor.Title,
			tractor.Price,
			tractor.OriginalPrice,
			tractor.PriceExclVAT,
			tractor.HP,
			tractor.Year,
			tractor.WorkingHours,
			tractor.Dealer,
			tractor.Location,
			tractor.ImageURL,
			tractor.Details,
		}
		if err := writer.Write(row); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("Results saved to %s\n", filename)
	fmt.Printf("Total tractors scraped: %d\n", len(tractors))
}