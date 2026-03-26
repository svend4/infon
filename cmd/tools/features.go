//go:build experimental
// +build experimental

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/svend4/infon/experimental/features"
)

func main() {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	enableCmd := flag.NewFlagSet("enable", flag.ExitOnError)
	disableCmd := flag.NewFlagSet("disable", flag.ExitOnError)
	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)

	// Флаги для list
	stabilityFilter := listCmd.String("stability", "", "Filter by stability (alpha, beta, candidate)")
	enabledOnly := listCmd.Bool("enabled", false, "Show only enabled features")

	// Флаги для info
	featureID := infoCmd.String("id", "", "Feature ID")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		listCmd.Parse(os.Args[2:])
		handleList(*stabilityFilter, *enabledOnly)

	case "enable":
		enableCmd.Parse(os.Args[2:])
		if len(enableCmd.Args()) < 1 {
			fmt.Println("Error: feature ID required")
			fmt.Println("Usage: features enable <feature-id>")
			os.Exit(1)
		}
		handleEnable(enableCmd.Args()[0])

	case "disable":
		disableCmd.Parse(os.Args[2:])
		if len(disableCmd.Args()) < 1 {
			fmt.Println("Error: feature ID required")
			fmt.Println("Usage: features disable <feature-id>")
			os.Exit(1)
		}
		handleDisable(disableCmd.Args()[0])

	case "info":
		infoCmd.Parse(os.Args[2:])
		if *featureID == "" {
			fmt.Println("Error: feature ID required")
			fmt.Println("Usage: features info -id <feature-id>")
			os.Exit(1)
		}
		handleInfo(*featureID)

	case "summary":
		handleSummary()

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("TVCP Experimental Features Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  features list [--stability=<alpha|beta|candidate>] [--enabled]")
	fmt.Println("  features info -id <feature-id>")
	fmt.Println("  features enable <feature-id>")
	fmt.Println("  features disable <feature-id>")
	fmt.Println("  features summary")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  features list --stability=beta")
	fmt.Println("  features info -id webrtc-processing")
	fmt.Println("  features enable advanced-aec")
	fmt.Println("  features summary")
}

func handleList(stabilityStr string, enabledOnly bool) {
	var featuresList []*features.Feature

	if stabilityStr != "" {
		var stability features.Stability
		switch stabilityStr {
		case "alpha":
			stability = features.Alpha
		case "beta":
			stability = features.Beta
		case "candidate":
			stability = features.Candidate
		default:
			fmt.Printf("Invalid stability: %s (use alpha, beta, or candidate)\n", stabilityStr)
			os.Exit(1)
		}
		featuresList = features.Registry.ListByStability(stability)
	} else if enabledOnly {
		featuresList = features.Registry.ListEnabled()
	} else {
		featuresList = features.Registry.List()
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Experimental Features                              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	if len(featuresList) == 0 {
		fmt.Println("No features found matching criteria")
		return
	}

	for i, f := range featuresList {
		status := "🔴"
		if f.Enabled {
			status = "🟢"
		}

		stabilityIcon := ""
		switch f.Stability {
		case features.Alpha:
			stabilityIcon = "⚠️  Alpha"
		case features.Beta:
			stabilityIcon = "🔶 Beta"
		case features.Candidate:
			stabilityIcon = "✅ Candidate"
		}

		fmt.Printf("%d. %s %s - %s (%s)\n", i+1, status, f.Name, f.ID, stabilityIcon)
		fmt.Printf("   %s\n", f.Description)
		fmt.Printf("   Performance: %s\n", f.PerformanceNote)
		fmt.Println()
	}

	fmt.Printf("Total: %d features\n", len(featuresList))
}

func handleInfo(featureID string) {
	feature, err := features.Registry.Get(featureID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	status := "Disabled 🔴"
	if feature.Enabled {
		status = "Enabled 🟢"
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  %s\n", feature.Name)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("ID:           %s\n", feature.ID)
	fmt.Printf("Status:       %s\n", status)
	fmt.Printf("Stability:    %s\n", feature.Stability)
	fmt.Printf("Version:      %s\n", feature.Version)
	fmt.Printf("Maintainer:   %s\n", feature.Maintainer)
	fmt.Println()
	fmt.Printf("Description:\n  %s\n", feature.Description)
	fmt.Println()
	fmt.Printf("Tests:        %d/%d passed (%.1f%% coverage)\n",
		feature.TestsPassed, feature.TestsTotal, feature.TestCoverage)
	fmt.Printf("Performance:  %s\n", feature.PerformanceNote)
	fmt.Println()

	if len(feature.Dependencies) > 0 {
		fmt.Println("Dependencies:")
		for _, dep := range feature.Dependencies {
			fmt.Printf("  - %s\n", dep)
		}
		fmt.Println()
	}

	if len(feature.Risks) > 0 {
		fmt.Println("⚠️  Risks:")
		for _, risk := range feature.Risks {
			fmt.Printf("  - %s\n", risk)
		}
		fmt.Println()
	}
}

func handleEnable(featureID string) {
	err := features.Registry.Enable(featureID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	feature, _ := features.Registry.Get(featureID)
	fmt.Printf("✅ Enabled: %s\n", feature.Name)

	if feature.Stability == features.Alpha {
		fmt.Println()
		fmt.Println("⚠️  WARNING: This is an Alpha feature!")
		fmt.Println("   It may be unstable and the API may change.")
	}

	if len(feature.Risks) > 0 {
		fmt.Println()
		fmt.Println("⚠️  Risks to be aware of:")
		for _, risk := range feature.Risks {
			fmt.Printf("   - %s\n", risk)
		}
	}
}

func handleDisable(featureID string) {
	err := features.Registry.Disable(featureID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	feature, _ := features.Registry.Get(featureID)
	fmt.Printf("🔴 Disabled: %s\n", feature.Name)
}

func handleSummary() {
	features.Registry.PrintSummary()

	// Статистика
	allFeatures := features.Registry.List()
	enabledFeatures := features.Registry.ListEnabled()
	alphaFeatures := features.Registry.ListByStability(features.Alpha)
	betaFeatures := features.Registry.ListByStability(features.Beta)
	candidateFeatures := features.Registry.ListByStability(features.Candidate)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Statistics                                                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Total features:     %d\n", len(allFeatures))
	fmt.Printf("Enabled:            %d\n", len(enabledFeatures))
	fmt.Printf("Disabled:           %d\n", len(allFeatures)-len(enabledFeatures))
	fmt.Println()
	fmt.Printf("Alpha:              %d\n", len(alphaFeatures))
	fmt.Printf("Beta:               %d\n", len(betaFeatures))
	fmt.Printf("Candidate:          %d\n", len(candidateFeatures))
	fmt.Println()

	// Общее покрытие тестами
	var totalTests, passedTests int
	for _, f := range allFeatures {
		totalTests += f.TestsTotal
		passedTests += f.TestsPassed
	}

	coverage := 0.0
	if totalTests > 0 {
		coverage = float64(passedTests) / float64(totalTests) * 100
	}

	fmt.Printf("Test coverage:      %.1f%% (%d/%d tests)\n", coverage, passedTests, totalTests)
}
