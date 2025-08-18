package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/spf13/cobra"
)

var force bool

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the notification cache",
	Long: `Remove all cached notifications and reset the cache to empty state.

This will delete all stored notification history. Use with caution as this
action cannot be undone. The next sync will treat all notifications as new.`,
	RunE: runClear,
}

func init() {
	clearCmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
}

func runClear(cmd *cobra.Command, args []string) error {
	// Load cache to check current state
	cache := cache.New(cacheDir)
	if err := cache.Load(cacheDir); err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	notifications := cache.GetNotifications()
	
	if len(notifications) == 0 {
		fmt.Println("Cache is already empty.")
		return nil
	}

	// Show confirmation unless forced
	if !force {
		fmt.Printf("This will clear %d cached notifications.\n", len(notifications))
		fmt.Print("Are you sure? [y/N]: ")
		
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Clear the cache
	cache.Clear()

	// Save empty cache
	if err := cache.Save(cacheDir); err != nil {
		return fmt.Errorf("failed to save cleared cache: %w", err)
	}

	fmt.Printf("✓ Cache cleared (%d notifications removed)\n", len(notifications))
	
	if verbose {
		fmt.Printf("Cache file updated: %s\n", cacheDir)
	}

	return nil
}