package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// checkHyprctl verifies that hyprctl is available
func checkHyprctl() error {
	_, err := exec.LookPath("hyprctl")
	if err != nil {
		return fmt.Errorf("hyprctl not found in PATH: %v", err)
	}
	return nil
}

func main() {
	// Define command line flags
	monitorName := flag.String("name", "", "Monitor name")
	workspaceRange := flag.String("range", "", "Workspace range (e.g., '1-5')")
	dryRun := flag.Bool("dry-run", false, "Print commands without executing them")
	verbose := flag.Bool("verbose", false, "Enable verbose output")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(
			os.Stderr,
			"  %s --name=monitor-name --range=start-end [options]\n\n",
			os.Args[0],
		)
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  %s --name=DP-1 --range=1-5\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Setup logging
	log.SetFlags(0) // Clean log output
	if *verbose {
		log.SetFlags(log.Ltime | log.Lmicroseconds)
	}

	// Validate input
	if *monitorName == "" || *workspaceRange == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Check for hyprctl
	if err := checkHyprctl(); err != nil {
		log.Fatal(err)
	}

	// Parse range
	rangeParts := strings.Split(*workspaceRange, "-")
	if len(rangeParts) != 2 {
		log.Fatal("Range must be in format 'n-m'")
	}

	start, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		log.Fatal("Invalid start range:", err)
	}

	end, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		log.Fatal("Invalid end range:", err)
	}

	// Build batch commands
	var batchCmds []string
	for ws := start; ws <= end; ws++ {
		batchCmds = append(batchCmds,
			fmt.Sprintf("keyword workspace %d,monitor:%s", ws, *monitorName),
			fmt.Sprintf("dispatch moveworkspacetomonitor %d %s", ws, *monitorName))
	}

	// Join all commands with semicolons
	batchCmd := strings.Join(batchCmds, ";")

	if *dryRun {
		fmt.Println("Would execute:")
		fmt.Printf("hyprctl --batch '%s'\n", batchCmd)
		return
	}

	if *verbose {
		log.Printf("Executing batch command for workspaces %d-%d on monitor %s",
			start, end, *monitorName)
	}

	// Execute all commands in a single batch
	cmd := exec.Command("hyprctl", "--batch", batchCmd)

	// Capture and display any error output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error executing batch commands: %v\nOutput: %s", err, output)
	}

	if *verbose {
		log.Printf("Successfully configured %d workspaces", end-start+1)
	}
}
