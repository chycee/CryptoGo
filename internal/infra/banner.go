package infra

import (
	"fmt"
	"strings"
)

// ANSI Color Codes
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
)

// PrintBanner displays the startup banner with mode-specific warnings
func PrintBanner(cfg *Config) {
	mode := strings.ToUpper(cfg.Trading.Mode)
	version := cfg.App.Version

	color := ColorGreen
	modeDesc := "SIMULATION"

	switch mode {
	case "REAL":
		color = ColorRed
		modeDesc = "REAL MONEY TRADING"
	case "DEMO":
		color = ColorYellow
		modeDesc = "TESTNET (PLAY MONEY)"
	case "PAPER":
		color = ColorCyan
		modeDesc = "INTERNAL SIMULATION"
	}

	fmt.Println()
	fmt.Printf("%s###########################################################%s\n", color, ColorReset)
	fmt.Printf("%s#                                                         #%s\n", color, ColorReset)
	fmt.Printf("%s#               🚀 CryptoGo Trading System                #%s\n", color, ColorReset)
	fmt.Printf("%s#                                                         #%s\n", color, ColorReset)
	fmt.Printf("%s#   MODE:    %-36s #%s\n", color, mode, ColorReset)
	fmt.Printf("%s#   TYPE:    %-36s #%s\n", color, modeDesc, ColorReset)
	fmt.Printf("%s#   VERSION: %-36s #%s\n", color, version, ColorReset)
	fmt.Printf("%s#                                                         #%s\n", color, ColorReset)

	if mode == "REAL" {
		fmt.Printf("%s#   ⚠️  WARNING: YOU ARE TRADING WITH REAL MONEY  ⚠️      #%s\n", ColorRed, ColorReset)
		fmt.Printf("%s#   ENSURE YOU HAVE VERIFIED YOUR STRATEGY IN DEMO        #%s\n", ColorRed, ColorReset)
	}

	fmt.Printf("%s###########################################################%s\n", color, ColorReset)
	fmt.Println()
}
