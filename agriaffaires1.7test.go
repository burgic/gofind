package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"encoding/base64"
	"strings"
	"time"
    

	"github.com/PuerkitoBio/goquery"
)

// Define the Tractor struct to hold the scraped data
type Tractor struct {
	DisplayedPrice     string
	// DisplayedCurrency  string
	// ReferencePrice     string
	// ReferenceCurrency  string
	PriceType          string
	Dealer             string
	Location           string
	PhoneNumber        string
	DebugInfo 		   map[string]string
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
    PriceExclVAT           string
    Comments               string
    // DisplayedPrice         string
    ReferencePrice         string
    ReferenceCurrency      string
    VATInfo                string
    Details                map[string]string
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
    req.Header.Set("X-Forwarded-For", "81.2.69.142")  // An example UK IP address
    req.Header.Set("CF-IPCountry", "GB")
    
    // Set a cookie to indicate UK preference
    req.AddCookie(&http.Cookie{Name: "country_code", Value: "gb"})
    
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    return resp, nil
}

// Scrape the page and extract the desired data
func scrapePage(url string) (*Tractor, error) {
    // ... (keep the existing setup code)

	resp, err := makeRequest(url)
    if err != nil {
        return nil, fmt.Errorf("error fetching page: %v", err)
    }
    defer resp.Body.Close()

    // Parse the response body using goquery
    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error parsing page: %v", err)
    }

    tractor := &Tractor{
        Details: make(map[string]string),
    }

    // Extract information from the table
    doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
        key := strings.TrimSpace(s.Find("td").First().Text())
        key = strings.TrimSuffix(key, ":")
        value := strings.TrimSpace(s.Find("td").Last().Text())
        
        // Store all information in Details map
        tractor.Details[key] = value

        // Also populate specific fields
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
            tractor.Power = value
        case "Dimension of front tires":
            tractor.FrontTireDimension = value
        case "Wear of front tires":
            tractor.FrontTireWear = value
        case "Wear of rear tires":
            tractor.RearTireWear = value
        case "Period of availability of spare parts":
            tractor.SparePartsAvailability = value
        case "Price excl. VAT":
            tractor.PriceExclVAT = value
        case "Comments":
            tractor.Comments = value
        }

        // Debug: Print each key-value pair as it's processed
        fmt.Printf("Processing: %s: %s\n", key, value)
    })

    // Extract the displayed price
    priceElement := doc.Find(".js-priceToChange").First()
    currencyElement := doc.Find(".js-currencyToChange").First()

    tractor.DisplayedPrice = strings.TrimSpace(priceElement.Text())
    displayedCurrency := strings.TrimSpace(currencyElement.Text())

    // Combine displayed price and currency
    if tractor.DisplayedPrice != "" && displayedCurrency != "" {
        tractor.DisplayedPrice += " " + displayedCurrency
    }

    // Reference price and currency from data attributes
    tractor.ReferencePrice, _ = priceElement.Attr("data-reference_price")
    tractor.ReferenceCurrency, _ = priceElement.Attr("data-reference_currency")

    // Extract VAT information
    vatElement := doc.Find(".price .h3-like.u-bold").First()
    tractor.VATInfo = strings.TrimSpace(vatElement.Text())

    // Add VAT info to displayed price if available
    if tractor.VATInfo != "" {
        tractor.DisplayedPrice += " " + tractor.VATInfo
    }

    // If displayed price is empty, use reference price
    if tractor.DisplayedPrice == "" {
        tractor.DisplayedPrice = tractor.ReferencePrice + " " + tractor.ReferenceCurrency
        if tractor.VATInfo != "" {
            tractor.DisplayedPrice += " " + tractor.VATInfo
        }
    }

	// Debug: Print all found table rows
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
    fmt.Printf("Row %d: %s\n", i, s.Text())
})
    return tractor, nil
}

func decodePhoneNumber(encoded string) string {
    decoded, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return encoded // Return the original string if decoding fails
    }
    // Remove the prefix (e.g., "(+45) ") if present
    decodedStr := string(decoded)
    parts := strings.SplitN(decodedStr, " ", 2)
    if len(parts) > 1 {
        return parts[1]
    }
    return decodedStr
}

func main() {
    url := "https://www.agriaffaires.co.uk/used/farm-tractor/44698339/fordson-major-super-major-med-trucktarn.html"
    
    tractor, err := scrapePage(url)
    if err != nil {
        log.Fatalf("Error scraping page: %v", err)
    }

    fmt.Println("Tractor Information:")
    fmt.Printf("Category: %s\n", tractor.Category)
    fmt.Printf("Type of ad: %s\n", tractor.AdType)
    fmt.Printf("Reference: %s\n", tractor.Reference)
    fmt.Printf("Make: %s\n", tractor.Make)
    fmt.Printf("Model: %s\n", tractor.Model)
    fmt.Printf("Status: %s\n", tractor.Status)
    fmt.Printf("Power: %s\n", tractor.Power)
    fmt.Printf("Front Tire Dimension: %s\n", tractor.FrontTireDimension)
    fmt.Printf("Front Tire Wear: %s\n", tractor.FrontTireWear)
    fmt.Printf("Rear Tire Wear: %s\n", tractor.RearTireWear)
    fmt.Printf("Spare Parts Availability: %s\n", tractor.SparePartsAvailability)
    fmt.Printf("Displayed Price: %s\n", tractor.DisplayedPrice)
    fmt.Printf("Reference Price: %s %s\n", tractor.ReferencePrice, tractor.ReferenceCurrency)

    fmt.Printf("Comments: %s\n", tractor.Comments)

    fmt.Println("\nAdditional Details:")
    for key, value := range tractor.Details {
       fmt.Printf("%s: %s\n", key, value)
    }
}