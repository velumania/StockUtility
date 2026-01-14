package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

func CompanyInfo(filePath, target string) (string, string, error) {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Iterate over the rows
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return "", "", fmt.Errorf("error reading CSV file: %w", err)
		}

		// Check if the first column matches the target string
		if len(record) > 2 && record[0] == target {
			return fmt.Sprintf(`%s`, record[1]), fmt.Sprintf(`%s`, record[2]), nil // Return values in the second and third columns
		}
	}

	// If no match is found, return 'NA' for both values
	return "NA", "NA", nil
}

type companyInfo struct {
	StockSymbol   string
	CompanyRatios string
}

func main() {
	sh_filePath := "/Users/velumani.a/Downloads/comapny_data_20022025_new.csv" // Path to your CSV file

	// Check for input and output file arguments
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s inputfile outputfile\n", os.Args[0])
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Open the input file
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open input file: %s\n", err)
	}
	defer file.Close()

	// Create the output file
	output, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %s\n", err)
	}
	defer output.Close()

	reader := csv.NewReader(file)
	writer := bufio.NewWriter(output)

	// Read all rows from the CSV
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Unable to parse file as CSV for %s: %v", file, err)
	}

	sequence := 1
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)

	// Loop through all rows, ignoring the first row and resetting the counter after 19 rows
	for index, row := range records {
		// Ignore the first row (header)
		if index == 0 {
			continue
		}

		// Check if the row has at least one column
		if len(row) > 0 {
			toReplace := []string{"-BE", "-EQ", "-BZ"}
			cleanedValue := row[0]

			for _, substr := range toReplace {
				cleanedValue = strings.ReplaceAll(cleanedValue, substr, "")
			}

			// fetch Shareholders data
			stockSymbol := cleanedValue

			// Fetch both values from the CompanyInfo method
			sh_value1, sh_value2, err := CompanyInfo(sh_filePath, stockSymbol)
			if err != nil {
				log.Fatalf("Error: %v", err)
				sh_value1 = "NA"
				sh_value2 = "NA"
				continue
			}

			// Replace special characters in row[0] with '_'
			trimmedText := re.ReplaceAllString(strings.TrimSpace(row[0]), "_")
			sanitizedValue := "NSE:" + strings.ReplaceAll(trimmedText, "_BE", "")

			// Format output with the fetched data
			pfret := "'" + row[1] + ", " + row[2] + ", " + sh_value1 + ", " + sh_value2 + "'"

			// Print and write in the desired format
			fmt.Printf("    index_%02d := '%s', index_%02d_mc := %s\n", sequence, strings.TrimSpace(sanitizedValue), sequence, pfret)
			_, err = fmt.Fprintf(writer, "    index_%02d := '%s', index_%02d_mc := %s\n", sequence, strings.TrimSpace(sanitizedValue), sequence, pfret)
			if err != nil {
				log.Fatalf("Failed to write to file: %s", err)
			}
		}

		sequence++
		if sequence > 19 {
			sequence = 1
			fmt.Printf("\n")
			_, err = fmt.Fprintf(writer, "\n")
			if err != nil {
				log.Fatalf("Failed to write to file: %s", err)
			}
		}
	}

	// Flush the writer to ensure all data is written to the file
	err = writer.Flush()
	if err != nil {
		log.Fatalf("Failed to flush writer: %s\n", err)
	}

	fmt.Println("Processing complete, output written to:", outputFile)
}
