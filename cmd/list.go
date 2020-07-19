package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var listNamespace string

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "show stuck pigeon queue",
	Long: `
Eg. pigeon-tool list -n all
Eg. pigeon-tool list -n NevecTW
	`,
	RunE: func(cmd *cobra.Command, args []string) error {

		var pigeon Information
		if staging {
			pigeon.pigeonHostEndpoint = "https://edge.dist.yahoo.com:4443/roles/v1/roles/nevec_egs_pigeon.HOSTs.int/members?output=json"
		} else {
			pigeon.pigeonHostEndpoint = "https://edge.dist.yahoo.com:4443/roles/v1/roles/nevec_egs_pigeon.HOSTs.prod/members?output=json"
		}
		pigeon.StatusURL = "/api/pigeon/v1/status"
		pigeon.cert = "/tmp/pigeon_admin_role.cert"

		client, err := getClient("")
		if err != nil {
			return err
		}
		hosts, err := doGet(client, pigeon.pigeonHostEndpoint)
		if err != nil {
			return fmt.Errorf("when getting Pigeon list %s: %s", pigeon.pigeonHostEndpoint, err.Error())
		}
		// get pigeon.HostList for later use
		if err = json.Unmarshal(hosts, &pigeon.HostList); err != nil {
			return fmt.Errorf("unmarshal fail for getting host list ")
		}
		// use role cert to call pigeon api
		roleClient, err := getClient(pigeon.cert)
		if err != nil {
			return err
		}

		responses := make(chan []byte)

		for _, host := range pigeon.HostList[0].Members {
			if strings.Contains(host, "tail") {
				pigeon.tailCount++
				pigeonStatus := fmt.Sprintf("https://%s:4443%s", host, pigeon.StatusURL)
				go doGetUseChan(roleClient, pigeonStatus, responses)
			}
		}

		for x := 0; x < pigeon.tailCount; x++ {
			if err = json.Unmarshal(<-responses, &pigeon.StatusResult); err != nil {
				return fmt.Errorf("unmarshal fail for getting pigeon api ")
			}

			for _, v := range pigeon.StatusResult.PigeonStatus.Sub {
				if listNamespace == "all" && v.OldMessageCount != 0 {
					fmt.Println()
					fmt.Println(pigeon.StatusResult.Host, v.Property, v.SubscriptionName)
					for _, id := range v.OldMessages {
						fmt.Println(id)
					}

				} else if v.OldMessageCount != 0 && v.Property == listNamespace {
					fmt.Println(pigeon.StatusResult.Host, v.Property, v.SubscriptionName)
					for _, id := range v.OldMessages {
						fmt.Println(id)
					}
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listNamespace, "namespace", "n", "", "namespace or all")
	listCmd.MarkFlagRequired("namespace")
}
