package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

var message string
var queue string

// skipCmd represents the skip command
var skipCmd = &cobra.Command{
	Use:   "skip",
	Short: "skip the certain message of queue or skip all messages of a queue",
	Long: `
Eg. pigeon-tool skip -q CQI.prod.storeeps.set.action::CQO.prod.storeeps.set.action.search.merlin -m d925d129-e4e7-4602-bba4-124bf462bc5c__08959ef907109ef601
Eg. pigeon-tool skip -q CQI.prod.storeeps.set.action::CQO.prod.storeeps.set.action.search.merlin -m all
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pigeon Information
		if staging {
			pigeon.pigeonHostEndpoint = "https://edge.dist.yahoo.com:4443/roles/v1/roles/nevec_egs_pigeon.HOSTs.int/members?output=json"
		} else {
			pigeon.pigeonHostEndpoint = "https://edge.dist.yahoo.com:4443/roles/v1/roles/nevec_egs_pigeon.HOSTs.prod/members?output=json"
		}
		pigeon.StatusURL = "/api/pigeon/v1/status"
		pigeon.SkipURL = "/api/pigeon/v1/messages/skip/"
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
		var wg sync.WaitGroup
		if message == "all" {
			// save message id to channel

			// call pigeon status api parallely
			for _, host := range pigeon.HostList[0].Members {
				if strings.Contains(host, "tail") {
					pigeon.tailCount++
					pigeonStatus := fmt.Sprintf("https://%s:4443%s", host, pigeon.StatusURL)
					go doGetUseChan(roleClient, pigeonStatus, responses)
				}
			}

			// get the result then parse the messageID
			for x := 0; x < pigeon.tailCount; x++ {
				if err = json.Unmarshal(<-responses, &pigeon.StatusResult); err != nil {
					return fmt.Errorf("unmarshal fail for getting pigeon api ")
				}
				for _, v := range pigeon.StatusResult.PigeonStatus.Sub {
					if v.SubscriptionName == queue && v.OldMessageCount != 0 {
						for _, id := range v.OldMessages {
							url := fmt.Sprintf("https://%s:4443%s%s?msgId=%s", pigeon.StatusResult.Host, pigeon.SkipURL, queue, id)
							wg.Add(1)
							go doPutUseChan(roleClient, url, nil, 200, &wg)

						}
						wg.Wait()
					}
				}
			}

		} else {
			for _, host := range pigeon.HostList[0].Members {
				if strings.Contains(host, "tail") {
					url := fmt.Sprintf("https://%s:4443%s%s?msgId=%s", host, pigeon.SkipURL, queue, message)
					doPut(roleClient, url, nil, 200)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(skipCmd)
	skipCmd.Flags().StringVarP(&queue, "queue", "q", "", "SubscriptionName")
	skipCmd.Flags().StringVarP(&message, "message", "m", "", "Message_id or [all]")
	skipCmd.MarkFlagRequired("queue")
	skipCmd.MarkFlagRequired("message")
}
