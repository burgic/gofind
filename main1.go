package main

import (
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
	url := "https://www.landwirt.com/en/used-farm-machinery/tractors.html"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
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

	// Create results directory if it doesn't exist
	err = os.MkdirAll("./results", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	// Create filename with current date and time
	now := time.Now()
	filename := filepath.Join("./results", fmt.Sprintf("tractor_results_%s.txt", now.Format("2006-01-02_15-04-05")))

	// Create and open the file
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Write results to the file
	for _, tractor := range tractors {
		fmt.Fprintf(file, "Title: %s\n", tractor.Title)
		fmt.Fprintf(file, "Price: %s\n", tractor.Price)
		if tractor.OriginalPrice != "" {
			fmt.Fprintf(file, "Original Price: %s\n", tractor.OriginalPrice)
		}
		fmt.Fprintf(file, "Price Excl. VAT: %s\n", tractor.PriceExclVAT)
		fmt.Fprintf(file, "HP: %s\n", tractor.HP)
		fmt.Fprintf(file, "Year: %s\n", tractor.Year)
		fmt.Fprintf(file, "Working Hours: %s\n", tractor.WorkingHours)
		fmt.Fprintf(file, "Dealer: %s\n", tractor.Dealer)
		fmt.Fprintf(file, "Location: %s\n", tractor.Location)
		fmt.Fprintf(file, "Image URL: %s\n", tractor.ImageURL)
		fmt.Fprintf(file, "Details: %s\n", tractor.Details)
		fmt.Fprintf(file, "------------------------\n")
	}

	fmt.Printf("Results saved to %s\n", filename)
}