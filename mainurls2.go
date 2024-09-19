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
	DetailURL     string
	Description   string
	Equipment     map[string]string
	Specifications map[string]string
}

const (
	baseDelay = 2 * time.Second
	jitter    = 2 * time.Second
)

func main() {
	baseURL := "https://www.landwirt.com/en/used-farm-machinery/used-McCormick-tractors.html"
	tractors := []Tractor{}

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?offset=%d", baseURL, (page-1)*20)
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

	doc.Find(".row.gmmtreffer").Each(func(i int, s *goquery.Selection) {
		tractor := Tractor{}

		tractor.Title = strings.TrimSpace(s.Find("h3 a").Text())
		tractor.DetailURL, _ = s.Find("h3 a").Attr("href")
		if !strings.HasPrefix(tractor.DetailURL, "http") {
			tractor.DetailURL = "https://www.landwirt.com" + tractor.DetailURL
		}
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

	// Extract equipment
	tractor.Equipment = make(map[string]string)
	doc.Find(".detail-equip .eitems").Each(func(i int, s *goquery.Selection) {
		key := strings.TrimSpace(s.Find("a").Text())
		if key == "" {
			key = strings.TrimSpace(s.Text())
		}
		value := "Yes"
		tractor.Equipment[key] = value
	})

	// Extract specifications
	tractor.Specifications = make(map[string]string)
	doc.Find(".detail-infos .row").Each(func(i int, s *goquery.Selection) {
		key := strings.TrimSpace(s.Find(".col-xs-6:first-child").Text())
		value := strings.TrimSpace(s.Find(".col-xs-6:last-child").Text())
		if key != "" && value != "" {
			tractor.Specifications[key] = value
		}
	})
}

func saveToCsv(tractors []Tractor) {
	err := os.MkdirAll("./results", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	filename := filepath.Join("./results", fmt.Sprintf("mccormick_tractor_results_%s.csv", now.Format("2006-01-02_15-04-05")))

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Collect all possible equipment and specification keys
	equipmentKeys := make(map[string]bool)
	specKeys := make(map[string]bool)
	for _, tractor := range tractors {
		for k := range tractor.Equipment {
			equipmentKeys[k] = true
		}
		for k := range tractor.Specifications {
			specKeys[k] = true
		}
	}

	// Create header
	header := []string{"Title", "Price", "Original Price", "Price Excl. VAT", "HP", "Year", "Working Hours", 
		"Dealer", "Location", "Image URL", "Details", "Detail URL", "Description"}
	for k := range equipmentKeys {
		header = append(header, "Equipment: "+k)
	}
	for k := range specKeys {
		header = append(header, "Spec: "+k)
	}
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
			tractor.DetailURL,
			tractor.Description,
		}
		for k := range equipmentKeys {
			if v, ok := tractor.Equipment[k]; ok {
				row = append(row, v)
			} else {
				row = append(row, "")
			}
		}
		for k := range specKeys {
			if v, ok := tractor.Specifications[k]; ok {
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