package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/AlvaroMrJack/k-flow-spec/internal/mock"
)

var port int

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Inicia un servidor local mock que simula la API de Kapso",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mock.NewServer(fmt.Sprintf(":%d", port))
		return server.Start()
	},
}

func init() {
	mockCmd.Flags().IntVarP(&port, "port", "p", 8080, "Puerto para el servidor mock")
	rootCmd.AddCommand(mockCmd)
}
