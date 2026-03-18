package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agnostikos",
	Short: "AgnosticOS CLI",
	Long:  `AgnosticOS is a tool for managing software packages across different operating systems.`,
	Version: "0.1.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.Flags().StringP("config", "c", "", "config file (default is $HOME/.agnostikos.yaml)")
}