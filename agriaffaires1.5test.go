package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"encoding/base64"
	"strings"
	"time"
    "math"
    "strconv"

	"github.com/PuerkitoBio/goquery"
)

// Define the Tractor struct to hold the scraped data
type Tractor struct {
	DisplayedPrice     string
	DisplayedCurrency  string
	ReferencePrice     string
	ReferenceCurrency  string
	PriceType          string
	Dealer             string
	Location           string
	PhoneNumber        string
	DebugInfo 		   map[string]string
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

    tractor := &Tractor{}

    // Extract the displayed price
    priceElement := doc.Find(".js-priceToChange").First()
    currencyElement := doc.Find(".js-currencyToChange").First()
    
    price := strings.TrimSpace(priceElement.Text())
    currency := strings.TrimSpace(currencyElement.Text())
    
    // Remove any thousand separators and convert to float
    price = strings.ReplaceAll(price, ",", "")
    priceFloat, err := strconv.ParseFloat(price, 64)
    if err == nil {
        // Round to nearest whole number
        priceFloat = math.Round(priceFloat)
        price = strconv.FormatFloat(priceFloat, 'f', 0, 64)
    }
    
    tractor.DisplayedPrice = price + " " + currency
    // Extract the displayed price (take the first non-empty value)
    doc.Find(".js-priceToChange").Each(func(i int, s *goquery.Selection) {
        if tractor.DisplayedPrice == "" {
            tractor.DisplayedPrice = strings.TrimSpace(s.Text())
        }
    })

    // Extract currency (take the first value)
    tractor.DisplayedCurrency = strings.TrimSpace(doc.Find(".js-currencyToChange").First().Text())

    // Combine price and currency
    if tractor.DisplayedPrice != "" && tractor.DisplayedCurrency != "" {
        tractor.DisplayedPrice = tractor.DisplayedPrice + " " + tractor.DisplayedCurrency
    }

    // Reference price and currency from data attributes
    tractor.ReferencePrice, _ = doc.Find(".js-priceToChange").First().Attr("data-reference_price")
    tractor.ReferenceCurrency, _ = doc.Find(".js-priceToChange").First().Attr("data-reference_currency")

    // Extract price type (e.g., "ex-VAT" or "inc-VAT")
    tractor.PriceType = strings.TrimSpace(strings.Split(doc.Find(".price .h3-like.u-bold").Text(), " ")[0])

    // Extract the dealer name
    tractor.Dealer = strings.TrimSpace(doc.Find(".block--contact-desktop .u-bold.h3-like.man").Text())

    // Extract the location (country)
    locationText := strings.TrimSpace(doc.Find(".block--contact-desktop .u-bold").Last().Text())
    locationParts := strings.Split(locationText, "\n")
    if len(locationParts) > 0 {
        tractor.Location = strings.TrimSpace(locationParts[len(locationParts)-1])
    }

    // Extract the phone number
    phoneElements := doc.Find(".js-hi-t")
    if phoneElements.Length() > 0 {
        tractor.PhoneNumber, _ = phoneElements.First().Attr("data-pdisplay")
        tractor.PhoneNumber = decodePhoneNumber(tractor.PhoneNumber)
    }

    // ... (keep the debug info code)

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

    fmt.Println("Extracted Information:")
    fmt.Printf("Displayed Price: %s\n", tractor.DisplayedPrice)
	fmt.Printf("Displayed Currency: %s\n", tractor.DisplayedCurrency)
    fmt.Printf("Reference Price: %s %s\n", tractor.ReferencePrice, tractor.ReferenceCurrency)
    fmt.Printf("Price Type: %s\n", tractor.PriceType)
    fmt.Printf("Dealer: %s\n", tractor.Dealer)
    fmt.Printf("Location: %s\n", tractor.Location)
    fmt.Printf("Phone Number: %s\n", tractor.PhoneNumber)

    fmt.Println("\nRaw HTML selectors:")
    for key, value := range tractor.DebugInfo {
        fmt.Printf("%s: %s\n", key, value)
    }
}