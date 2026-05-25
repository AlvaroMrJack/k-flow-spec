package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/AlvaroMrJack/k-flow-spec/internal/logger"
	"github.com/AlvaroMrJack/k-flow-spec/internal/signal"
)

var RootCmd = &cobra.Command{
	Use:   "kfs",
	Short: "QA automatizado para flujos WhatsApp",
	Long:  `k-flow-spec (kfs) es un CLI spec-driven para testear workflows de WhatsApp.`,
}

func Execute() {
	logger.Setup()
	ctx := signal.SetupContext(context.Background())
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
