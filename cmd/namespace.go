package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var namespace = &cobra.Command{
	Use:   "ns-list",
	Short: "list all namespace pigeon use",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var ns string = `  
		AdpostTW
		AuctionsHK
		AuctionsTW
		BillingTW
		DataMiningTW
		ECCentralTech
		NevecTW
		ShoppingMall
		Store-TW
		`
		fmt.Println(ns)
	},
}

func init() {
	rootCmd.AddCommand(namespace)
}
