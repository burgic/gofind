package main

import (
    "fmt"
    "github.com/playwright-community/playwright-go"
)

func main() {
    // Start Playwright
    pw, err := playwright.Run()
    if err != nil {
        panic(err)
    }
    defer pw.Stop()

    // Launch a new browser
    browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
        Headless: playwright.Bool(false), // Set to true to run headless
    })
    if err != nil {
        panic(err)
    }
    defer browser.Close()

    // Open a new page
    page, err := browser.NewPage()
    if err != nil {
        panic(err)
    }

    // Navigate to the tractor page
    _, err = page.Goto("https://www.agriaffaires.co.uk/used/farm-tractor/44698339/fordson-major-super-major-med-trucktarn.html")
    if err != nil {
        panic(err)
    }

    // Wait for the page to load fully (network idle state)
    err = page.WaitForLoadState(playwright.LoadStateNetworkIdle)
    if err != nil {
        panic(err)
    }

    // Extract the price information
    priceElement, err := page.QuerySelector(".js-priceToChange")
    if err != nil {
        panic(err)
    }

    // Get the displayed price
    displayedPrice, err := priceElement.InnerText()
    if err != nil {
        panic(err)
    }

    // Get the reference price and currency from attributes
    referencePrice, _ := priceElement.GetAttribute("data-reference_price")
    referenceCurrency, _ := priceElement.GetAttribute("data-reference_currency")

    fmt.Printf("Reference Price: %s\n", referencePrice)
    fmt.Printf("Reference Currency: %s\n", referenceCurrency)
    fmt.Printf("Displayed Price: %s\n", displayedPrice)
}
