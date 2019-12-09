// Copyright (c) 2019 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"sort"

	"github.com/projectcalico/felix/bpf"

	"github.com/projectcalico/felix/bpf/routes"
	"github.com/projectcalico/felix/ip"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	routesCmd.AddCommand(routesDumpCmd)
	rootCmd.AddCommand(routesCmd)
}

var routesDumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "dumps routes",
	Run: func(cmd *cobra.Command, args []string) {
		if err := dumpRoutes(); err != nil {
			log.WithError(err).Error("Failed to dump routes map.")
		}
	},
}

// routesCmd represents the routes command
var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Manipulates routes",
}

func dumpRoutes() error {
	mc := &bpf.MapContext{}
	routesMap := routes.Map(mc)

	var dests []ip.CIDR
	valueByDest := map[ip.CIDR]routes.Value{}

	err := routesMap.Iter(func(k, v []byte) {
		var key routes.Key
		var value routes.Value
		copy(key[:], k)
		copy(value[:], v)

		dest := key.Dest()
		valueByDest[dest] = value
		dests = append(dests, dest)
	})
	if err != nil {
		return err
	}

	sortCIDRs(dests)

	for _, dest := range dests {
		var detail string
		v := valueByDest[dest]
		switch v.Type() {
		case routes.TypeRemoteWorkload:
			detail = fmt.Sprintf("remote workload, host IP %v", v.NextHop())
		case routes.TypeRemoteHost:
			detail = "remote host"
		case routes.TypeLocalHost:
			detail = "local host"
		case routes.TypeLocalWorkload:
			detail = "local workload"
		case routes.TypeUnknown:
			fallthrough
		default:
			detail = fmt.Sprintf("unknown %v", v)
		}
		fmt.Printf("%15v: %s\n", dest, detail)
	}

	return nil
}

func sortCIDRs(cidrs []ip.CIDR) {
	sort.Slice(cidrs, func(i, j int) bool {
		addrA := cidrs[i].Addr().(ip.V4Addr) // FIXME IPv6
		addrB := cidrs[j].Addr().(ip.V4Addr)
		for byteIdx := 0; byteIdx < 4; byteIdx++ {
			if addrA[byteIdx] < addrB[byteIdx] {
				return true
			}
			if addrA[byteIdx] > addrB[byteIdx] {
				return false
			}
		}
		return cidrs[i].Prefix() < cidrs[j].Prefix()
	})
}
