package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/logger"
	"github.com/AlvaroMrJack/k-flow-spec/internal/signal"
)

var rootCmd = &cobra.Command{
	Use:   "kfs",
	Short: "QA automatizado para flujos WhatsApp en Kapso",
	Long:  `k-flow-spec (kfs) es un CLI spec-driven para testear workflows de Kapso.`,
}

func main() {
	// Setup central logger
	logger.Setup()

	// Create a context that listens for SIGINT/SIGTERM
	ctx := signal.SetupContext(context.Background())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
