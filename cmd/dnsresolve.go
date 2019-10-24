// Copyright 2017 Prospect One https://prospectone.io/. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ProspectOne/perfops-cli/cmd/internal"
	"github.com/ProspectOne/perfops-cli/perfops"
)

var (
	dnsResolveCmd = &cobra.Command{
		Use:     "resolve [target]",
		Short:   "Resolve a DNS record on a domain name",
		Long:    `Resolve a DNS record on a target, e.g., google.com.`,
		Example: `perfops resolve --dns-server 8.8.8.8 --type A bing.com`,
		Args:    requireTarget(),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newPerfOpsClient()
			if err != nil {
				return err
			}
			return chkRunError(runDNSResolve(c, args[0], dnsResolveType, dnsResolveDNSServer, from, nodeIDs, dnsResolveLimit))
		},
	}

	dnsResolveType      string
	dnsResolveDNSServer string
	dnsResolveLimit     int
)

func initDNSResolveCmd(parentCmd *cobra.Command) {
	addCommonFlags(dnsResolveCmd)

	dnsResolveCmd.Flags().StringVarP(&dnsResolveType, "type", "T", "", "The DNS query type. On of: A, AAAA, CNAME, MX, NAPTR, NS, PTR, SOA, SPF, SRV, TXT.")
	dnsResolveCmd.Flags().StringVarP(&dnsResolveDNSServer, "dns-server", "S", "", "The DNS server to use to query for the test. You can use 127.0.0.1 to use the local resolver for location based benchmarking.")
	dnsResolveCmd.Flags().IntVarP(&dnsResolveLimit, "limit", "L", 1, "The maximum number of nodes to use")

	dnsResolveCmd.MarkFlagRequired("type")
	dnsResolveCmd.MarkFlagRequired("dns-server")

	parentCmd.AddCommand(dnsResolveCmd)
}

func runDNSResolve(c *perfops.Client, target, queryType, dnsServer, from string, nodeIDs []int, limit int) error {
	ctx := context.Background()
	dnsResolveReq := &perfops.DNSResolveRequest{
		Target:    target,
		Param:     queryType,
		DNSServer: dnsServer,
		Location:  from,
		Nodes:     nodeIDs,
		Limit:     limit,
	}

	spinner := internal.NewSpinner()
	fmt.Println("")
	spinner.Start()

	testID, err := c.Run.DNSResolve(ctx, dnsResolveReq)
	spinner.Stop()
	if err != nil {
		return err
	}

	if debug && !outputJSON {
		fmt.Printf("Test ID: %v\n", testID)
	}

	var output *perfops.DNSTestOutput
	printedIDs := map[string]bool{}
	for {
		spinner.Start()
		select {
		case <-time.After(500 * time.Millisecond):
		}

		output, err = c.Run.DNSResolveOutput(ctx, testID)
		spinner.Stop()
		if err != nil {
			return err
		}

		if !outputJSON {
			printPartialDNSOutput(fmt.Printf, output, printedIDs, func(r *perfops.DNSTestResult) string {
				o := r.ResolveOutput()
				return strings.Join(o, "\n")
			})
		}
		if output.IsFinished() {
			break
		}
	}
	if outputJSON {
		internal.PrintOutputJSON(output)
	}
	return nil
}

func printPartialDNSOutput(printf func(format string, a ...interface{}) (n int, err error), output *perfops.DNSTestOutput, printedIDs map[string]bool, getOutput func(r *perfops.DNSTestResult) string) {
	for _, item := range output.Items {
		if printedIDs[item.ID] {
			continue
		}
		r := item.Result
		n := r.Node
		if r.Message == "" {
			printedIDs[item.ID] = true
			o := getOutput(r)
			if o == "-2" {
				o = "The command timed-out. It either took too long to execute or we could not connect to your target at all."
			}
			printf("Node%d, AS%d, %s, %s\n%s\n", n.ID, n.AsNumber, n.City, n.Country.Name, o)
		} else if r.Message != "NO DATA" {
			printedIDs[item.ID] = true
			printf("Node%d, AS%d, %s, %s\n%s\n", n.ID, n.AsNumber, n.City, n.Country.Name, r.Message)
		}
	}
}
