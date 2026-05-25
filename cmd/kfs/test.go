package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "kfs generate + kfs run en un solo comando",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Ejecutando kfs generate...")
		generateCmd.RunE(generateCmd, args)

		fmt.Println("\nEjecutando kfs run...")
		runCmd.RunE(runCmd, args)

		return nil
	},
}

func runKfsCommand(subcmd string, args ...string) error {
	c := exec.Command("kfs", append([]string{subcmd}, args...)...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func init() {
	rootCmd.AddCommand(testCmd)
}
