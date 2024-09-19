package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Tractor struct {
	Title       string
	Price       string
	HP          string
	Year        string
	WorkingHours string
	Dealer      string
	Location    string
	ImageURL    string
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
		tractor.ImageURL, _ = s.Find(".bildboxgmm img").Attr("src")

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

		dealerInfo := strings.TrimSpace(s.Find("address").Text())
		parts := strings.Split(dealerInfo, "-")
		if len(parts) == 2 {
			tractor.Dealer = strings.TrimSpace(parts[0])
			tractor.Location = strings.TrimSpace(parts[1])
		}

		tractors = append(tractors, tractor)
	})

	for _, tractor := range tractors {
		fmt.Printf("Title: %s\n", tractor.Title)
		fmt.Printf("Price: %s\n", tractor.Price)
		fmt.Printf("HP: %s\n", tractor.HP)
		fmt.Printf("Year: %s\n", tractor.Year)
		fmt.Printf("Working Hours: %s\n", tractor.WorkingHours)
		fmt.Printf("Dealer: %s\n", tractor.Dealer)
		fmt.Printf("Location: %s\n", tractor.Location)
		fmt.Printf("Image URL: %s\n", tractor.ImageURL)
		fmt.Println("------------------------")
	}
}