package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/bin-ke/my-notion/internal/ai"
)

func main() {
	aiConfig := ai.LoadConfig()
	if aiConfig.APIKey == "" {
		log.Fatal("DEEPSEEK_API_KEY not set")
	}

	client := ai.NewClient(aiConfig)
	if client == nil {
		log.Fatal("failed to create AI client")
	}

	harness := ai.NewEvalHarness()

	// Optional: load custom cases
	if len(os.Args) > 1 {
		if err := harness.LoadCases(os.Args[1]); err != nil {
			log.Fatalf("failed to load cases: %v", err)
		}
	}

	category := ""
	if len(os.Args) > 2 {
		category = os.Args[2]
	}

	fmt.Println("Running AI evaluation...")
	results := harness.Run(client, category)

	if len(results) == 0 {
		fmt.Println("No cases matched the filter.")
		return
	}

	// Print summary
	metrics := harness.Metrics()
	summary, _ := json.MarshalIndent(metrics, "", "  ")
	fmt.Println(string(summary))

	// Save results
	if err := harness.ExportResults("eval-results.json"); err != nil {
		log.Printf("WARNING: failed to export results: %v", err)
	} else {
		fmt.Println("Results saved to eval-results.json")
	}
}
