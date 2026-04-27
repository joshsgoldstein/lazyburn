package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joshsgoldstein/lazyburn/internal/pricing"
	"github.com/spf13/cobra"
)

const pricingURL = "https://raw.githubusercontent.com/joshsgoldstein/lazyburn/main/pricing.json"

var updatePricingCmd = &cobra.Command{
	Use:   "update-pricing",
	Short: "Fetch the latest model pricing from GitHub",
	Long: `Download the latest pricing.json from the lazyburn repository and cache it locally.

Cached pricing is used on all future runs. Falls back to compiled-in defaults
if the cache is missing or unreadable.

Cache location: ~/.claude/lazyburn/pricing.json`,
	RunE: runUpdatePricing,
}

func init() {
	rootCmd.AddCommand(updatePricingCmd)
}

func runUpdatePricing(cmd *cobra.Command, args []string) error {
	fmt.Printf("Fetching pricing from %s\n", pricingURL)

	resp, err := http.Get(pricingURL)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	// Validate before saving.
	var pf pricing.PricingFile
	if err := json.Unmarshal(body, &pf); err != nil {
		return fmt.Errorf("invalid pricing JSON: %w", err)
	}
	if len(pf.Models) == 0 {
		return fmt.Errorf("pricing file contains no models")
	}

	cachePath := pricing.CachePath()
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("could not create cache directory: %w", err)
	}
	if err := os.WriteFile(cachePath, body, 0644); err != nil {
		return fmt.Errorf("could not write cache: %w", err)
	}

	fmt.Printf("Pricing updated (%s)\n\n", pf.Updated)
	fmt.Printf("%-22s  %6s  %8s  %8s  %9s  %7s\n", "Model", "Input", "Cache 5m", "Cache 1h", "Cache Read", "Output")
	fmt.Printf("%-22s  %6s  %8s  %8s  %9s  %7s\n", "-----", "-----", "--------", "--------", "----------", "------")
	for model, e := range pf.Models {
		fmt.Printf("%-22s  %6.2f  %8.2f  %8.2f  %9.2f  %7.2f\n",
			model, e.Input, e.Cache5m, e.Cache1h, e.CacheRead, e.Output)
	}
	fmt.Printf("\nPer million tokens. Cached at %s\n", cachePath)
	return nil
}
