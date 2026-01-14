package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type CompanyData struct {
	StockSymbol         string
	CompanyRatios       string
	ShareholdingDetails []string
}

// Function to transform keys based on the provided mappings
func transformKey(key string) string {
	transformations := map[string]string{
		"Market Cap":     "MktCap",
		"Current Price":  "CP",
		"High / Low":     "HL",
		"Stock P/E":      "P/E",
		"Book Value":     "BV",
		"Dividend Yield": "DY",
		"ROCE":           "ROCE",
		"ROE":            "ROE",
		"Face Value":     "FV",
	}

	trimmedKey := strings.TrimSpace(key)
	if newKey, exists := transformations[trimmedKey]; exists {
		return newKey
	}
	return trimmedKey
}

// Function to fetch and transform company ratios and shareholding details from Screener.in
func fetchCompanyData(stockSymbol string) (CompanyData, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Set up a timeout to prevent hanging
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var ratios []string
	var shareholders []string

	url := fmt.Sprintf("https://www.screener.in/company/%s/consolidated/", stockSymbol)

	js_ratios := `
    Array.from(document.querySelectorAll("div.company-ratios li")).map(li => {
        let name = li.querySelector("span.name") ? li.querySelector("span.name").innerText : "";
        let numbers = Array.from(li.querySelectorAll("span.number")).map(span => 
            span.innerText.replace(/,/g, "") // Remove commas from numbers
        );
        return name + ": " + numbers.join(" / ");
    });
`
	js_shp := `Array.from(document.querySelector('#quarterly-shp table.data-table tbody').querySelectorAll('tr'))
	.slice(0, Math.min(4, document.querySelector('#quarterly-shp table.data-table tbody').querySelectorAll('tr').length)) // Avoid out-of-bounds error
	.map(row => {
		let cells = row.querySelectorAll('td');
		return cells.length > 12 ? cells[12].innerText : (cells.length > 1 ? cells[1].innerText : "0.0%"); // Handle missing cells safely
	})
`
	//	js_shp := `Array.from(document.querySelector('#quarterly-shp table.data-table tbody').querySelectorAll('tr'))
	//    .slice(0, 4).map(row => {
	//        let cell = row.querySelectorAll('td')[12];
	//        return cell ? cell.innerText : row.querySelectorAll('td')[1].innerText;
	//    })
	//`
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`div.company-ratios`, chromedp.ByQuery),
		chromedp.WaitReady(`div.company-ratios`, chromedp.ByQuery), // Allow content load
		chromedp.Sleep(2*time.Second),                              // Allow content to load
		chromedp.Evaluate(js_shp, &shareholders),
		chromedp.Evaluate(js_ratios, &ratios),
	)
	if err != nil {
		return CompanyData{}, fmt.Errorf("error fetching company data for %s: %w", stockSymbol, err)
	}
	// Ensure ShareholdingDetails has at least 4 elements to avoid index out of range
	for len(shareholders) < 4 {
		shareholders = append(shareholders, "0.0%") // Fill missing entries with default value
	}
	// Transform keys and join all ratios into a single string, skipping certain keys
	var transformedRatios []string
	for _, ratio := range ratios {
		parts := strings.SplitN(ratio, ":", 2)
		if len(parts) != 2 {
			continue // Skip invalid entries
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		transformedKey := transformKey(key)

		// Skip specific keys
		if transformedKey == "CP" || transformedKey == "HL" || transformedKey == "DY" {
			continue
		}

		// Determine the separator based on the key
		separator := "; "
		if transformedKey == "MktCap" {
			separator = ", "
		}

		// Format and append the transformed ratio using the determined separator
		transformedRatios = append(transformedRatios, fmt.Sprintf("%s:%s%s", transformedKey, value, separator))
	}

	// Remove trailing separators from the final string
	finalRatios := strings.TrimRight(strings.Join(transformedRatios, ""), ",;")

	return CompanyData{
		StockSymbol:         stockSymbol,
		CompanyRatios:       finalRatios,
		ShareholdingDetails: shareholders,
	}, nil
}

func main() {
	inputFile, err := os.Open("/Users/velumani.a/Downloads/stock_symbols_19022025_new-set2.csv")
	if err != nil {
		log.Fatalf("Failed to open input CSV file: %v", err)
	}
	defer inputFile.Close()

	reader := csv.NewReader(inputFile)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV file: %v", err)
	}

	outputFile, err := os.Create("/Users/velumani.a/Downloads/comapny_data_19022025_new-set2.csv")
	if err != nil {
		log.Fatalf("Failed to create output CSV file: %v", err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// Write header to output CSV file
	writer.Write([]string{"Stock Symbol", "Shareholding Details", "Company Ratios"})

	for _, record := range records {
		stockSymbol := record[0]
		fmt.Printf("Fetching data for stock: %s\n", stockSymbol)

		// Fetch and transform company data for the stock
		data, err := fetchCompanyData(stockSymbol)
		if err != nil {
			log.Printf("Error fetching data for %s: %v", stockSymbol, err)
			continue
		}

		// Format shareholding details and handle special case
		formattedShareholding := []string{
			fmt.Sprintf("P:%s", strings.TrimSpace(data.ShareholdingDetails[0])),
			fmt.Sprintf("FIIs:%s", strings.TrimSpace(data.ShareholdingDetails[1])),
			fmt.Sprintf("DIIs:%s", strings.TrimSpace(data.ShareholdingDetails[2])),
			fmt.Sprintf("O:%s", func(value string) string {
				if strings.Contains(value, ",") {
					return "0.0%"
				}
				return strings.TrimSpace(value)
			}(data.ShareholdingDetails[3])),
		}
		writer.Write([]string{
			data.StockSymbol,
			strings.Join(formattedShareholding, "; "),
			data.CompanyRatios,
		})
	}

	fmt.Println("Data fetching and transformation complete. Output written to unmatched-shp-crs.csv")
}
