// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipv4_entry_test

import (
	"context"
	"testing"
	"time"

	"github.com/openconfig/featureprofiles/internal/attrs"
	"github.com/openconfig/featureprofiles/internal/deviations"
	"github.com/openconfig/featureprofiles/internal/fptest"
	"github.com/openconfig/gribigo/chk"
	"github.com/openconfig/gribigo/client"
	"github.com/openconfig/gribigo/constants"
	"github.com/openconfig/gribigo/fluent"
	"github.com/openconfig/ondatra"
)

const (
	// Next-hop group ID for dstPfx
	nhgID = 42
	// Next-hop 1 ID for dutPort2
	nh1ID = 43
	// Next-hop 2 ID for dutPort3
	nh2ID = 44
	// Unconfigured next-hop ID
	badNH = 45
	// Unconfigured static MAC address
	badMAC = "02:00:00:00:00:01"
)

const (
	// Destination prefix for DUT to ATE traffic.
	dstPfx      = "198.51.100.0/24"
	dstPfxMin   = "198.51.100.0"
	dstPfxMax   = "198.51.100.255"
	dstPfxCount = 256
)

var (
	dutPort1 = attrs.Attributes{
		Desc:    "DUT Port 1",
		IPv4:    "192.0.2.1",
		IPv4Len: 30,
	}
	dutPort2 = attrs.Attributes{
		Desc:    "DUT Port 2",
		IPv4:    "192.0.2.5",
		IPv4Len: 30,
	}
	dutPort3 = attrs.Attributes{
		Desc:    "DUT Port 3",
		IPv4:    "192.0.2.9",
		IPv4Len: 30,
	}

	atePort1 = attrs.Attributes{
		Name:    "atePort1",
		Desc:    "ATE Port 1",
		IPv4:    "192.0.2.2",
		IPv4Len: 30,
	}
	atePort2 = attrs.Attributes{
		Name:    "atePort2",
		Desc:    "ATE Port 2",
		IPv4:    "192.0.2.6",
		IPv4Len: 30,
	}
	atePort3 = attrs.Attributes{
		Name:    "atePort3",
		Desc:    "ATE Port 3",
		IPv4:    "192.0.2.10",
		IPv4Len: 30,
	}
)

func TestMain(m *testing.M) {
	fptest.RunTests(m)
}

// TestIPv4Entry tests a single IPv4Entry forwarding entry.
func TestIPv4Entry(t *testing.T) {
	ctx := context.Background()

	dut := ondatra.DUT(t, "dut")
	configureDUT(t, dut)

	ate := ondatra.ATE(t, "ate")
	ateTop := configureATE(t, ate)

	port2Flow := createFlow("Port 1 to Port 2", ate, ateTop, &atePort2)
	port3Flow := createFlow("Port 1 to Port 3", ate, ateTop, &atePort3)
	ecmpFlow := createFlow("Port 1 to Port 2 & 3", ate, ateTop, &atePort2, &atePort3)

	cases := []struct {
		desc                 string
		entries              []fluent.GRIBIEntry
		downPort             *ondatra.Port
		wantGoodFlows        []*ondatra.Flow
		wantBadFlows         []*ondatra.Flow
		wantOperationResults []*client.OpResult
	}{
		{
			desc: "Single next-hop",
			entries: []fluent.GRIBIEntry{
				fluent.NextHopEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithIndex(nh1ID).WithIPAddress(atePort2.IPv4),
				fluent.NextHopGroupEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithID(nhgID).AddNextHop(nh1ID, 1),
				fluent.IPv4Entry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithPrefix(dstPfx).WithNextHopGroup(nhgID),
			},
			wantGoodFlows: []*ondatra.Flow{port2Flow},
			wantBadFlows:  []*ondatra.Flow{port3Flow},
			wantOperationResults: []*client.OpResult{
				fluent.OperationResult().
					WithNextHopOperation(nh1ID).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithNextHopGroupOperation(nhgID).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithIPv4Operation(dstPfx).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
			},
		},
		{
			desc: "Multiple next-hops",
			entries: []fluent.GRIBIEntry{
				fluent.NextHopEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithIndex(nh1ID).WithIPAddress(atePort2.IPv4),
				fluent.NextHopEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithIndex(nh2ID).WithIPAddress(atePort3.IPv4),
				fluent.NextHopGroupEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithID(nhgID).AddNextHop(nh1ID, 1).AddNextHop(nh2ID, 1),
				fluent.IPv4Entry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithPrefix(dstPfx).WithNextHopGroup(nhgID),
			},
			wantGoodFlows: []*ondatra.Flow{ecmpFlow},
			wantOperationResults: []*client.OpResult{
				fluent.OperationResult().
					WithNextHopOperation(nh1ID).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithNextHopOperation(nh2ID).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithNextHopGroupOperation(nhgID).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithIPv4Operation(dstPfx).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
			},
		},
		{
			desc: "Nonexistant next-hop",
			entries: []fluent.GRIBIEntry{
				fluent.NextHopGroupEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithID(nhgID).AddNextHop(badNH, 1),
				fluent.IPv4Entry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithPrefix(dstPfx).WithNextHopGroup(nhgID),
			},
			wantBadFlows: []*ondatra.Flow{port2Flow, port3Flow},
			wantOperationResults: []*client.OpResult{
				fluent.OperationResult().
					WithNextHopGroupOperation(nhgID).
					WithProgrammingResult(fluent.ProgrammingFailed).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithIPv4Operation(dstPfx).
					WithProgrammingResult(fluent.ProgrammingFailed).
					WithOperationType(constants.Add).
					AsResult(),
			},
		},
		{
			desc:     "Downed next-hop interface",
			downPort: ate.Port(t, "port2"),
			entries: []fluent.GRIBIEntry{
				fluent.NextHopEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithIndex(nh1ID).WithIPAddress(atePort2.IPv4).
					WithInterfaceRef(dut.Port(t, "port2").Name()).WithMacAddress(badMAC),
				fluent.NextHopGroupEntry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithID(nhgID).AddNextHop(nh1ID, 1),
				fluent.IPv4Entry().WithNetworkInstance(*deviations.DefaultNetworkInstance).
					WithPrefix(dstPfx).WithNextHopGroup(nhgID),
			},
			wantBadFlows: []*ondatra.Flow{port2Flow, port3Flow},
			wantOperationResults: []*client.OpResult{
				fluent.OperationResult().
					WithNextHopOperation(nh1ID).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithNextHopGroupOperation(nhgID).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
				fluent.OperationResult().
					WithIPv4Operation(dstPfx).
					WithProgrammingResult(fluent.InstalledInFIB).
					WithOperationType(constants.Add).
					AsResult(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			gribic := dut.RawAPIs().GRIBI().Default(t)
			c := fluent.NewClient()
			c.Connection().WithStub(gribic).
				WithRedundancyMode(fluent.ElectedPrimaryClient).
				WithFIBACK().
				WithInitialElectionID(1, 0)
			c.Start(ctx, t)
			defer c.Stop(t)
			c.StartSending(ctx, t)
			if err := awaitTimeout(ctx, c, t, time.Minute); err != nil {
				t.Fatalf("Await got error during session negotiation: %v", err)
			}

			if tc.downPort != nil {
				ate.Actions().NewSetPortState().WithPort(tc.downPort).WithEnabled(false).Send(t)
				defer ate.Actions().NewSetPortState().WithPort(tc.downPort).WithEnabled(true).Send(t)
			}

			c.Modify().AddEntry(t, tc.entries...)
			if err := awaitTimeout(ctx, c, t, time.Minute); err != nil {
				t.Fatalf("Await got error for entries: %v", err)
			}
			defer func() {
				c.Modify().DeleteEntry(t, tc.entries...)
				if err := awaitTimeout(ctx, c, t, time.Minute); err != nil {
					t.Fatalf("Await got error for entries: %v", err)
				}
			}()

			for _, wantResult := range tc.wantOperationResults {
				chk.HasResult(t, c.Results(t), wantResult, chk.IgnoreOperationID())
			}

			validateTrafficFlows(t, ate, tc.wantGoodFlows, tc.wantBadFlows)
		})
	}
}

// configureDUT configures port1-3 on the DUT.
func configureDUT(t *testing.T, dut *ondatra.DUTDevice) {
	d := dut.Config()

	p1 := dut.Port(t, "port1")
	p2 := dut.Port(t, "port2")
	p3 := dut.Port(t, "port3")

	d.Interface(p1.Name()).Replace(t, dutPort1.NewInterface(p1.Name()))
	d.Interface(p2.Name()).Replace(t, dutPort2.NewInterface(p2.Name()))
	d.Interface(p3.Name()).Replace(t, dutPort3.NewInterface(p3.Name()))
}

// configreATE configures port1-3 on the ATE.
func configureATE(t *testing.T, ate *ondatra.ATEDevice) *ondatra.ATETopology {
	top := ate.Topology().New()

	p1 := ate.Port(t, "port1")
	p2 := ate.Port(t, "port2")
	p3 := ate.Port(t, "port3")

	atePort1.AddToATE(top, p1, &dutPort1)
	atePort2.AddToATE(top, p2, &dutPort2)
	atePort3.AddToATE(top, p3, &dutPort3)

	top.Push(t).StartProtocols(t)

	return top
}

// createFlow returns a flow from atePort1 to the dstPfx, expected to arrive on ATE interface dsts.
func createFlow(name string, ate *ondatra.ATEDevice, ateTop *ondatra.ATETopology, dsts ...*attrs.Attributes) *ondatra.Flow {
	hdr := ondatra.NewIPv4Header()
	hdr.WithSrcAddress(dutPort1.IPv4).
		DstAddressRange().WithMin(dstPfxMin).WithMax(dstPfxMax).WithCount(dstPfxCount)

	endpoints := []ondatra.Endpoint{}
	for _, dst := range dsts {
		endpoints = append(endpoints, ateTop.Interfaces()[dst.Name])
	}

	flow := ate.Traffic().NewFlow(name).
		WithSrcEndpoints(ateTop.Interfaces()[atePort1.Name]).
		WithDstEndpoints(endpoints...).
		WithHeaders(ondatra.NewEthernetHeader(), hdr)

	return flow
}

func validateTrafficFlows(t *testing.T, ate *ondatra.ATEDevice, good []*ondatra.Flow, bad []*ondatra.Flow) {
	if len(good) == 0 && len(bad) == 0 {
		return
	}

	flows := append(good, bad...)
	ate.Traffic().Start(t, flows...)
	time.Sleep(15 * time.Second)
	ate.Traffic().Stop(t)

	for _, flow := range good {
		if got := ate.Telemetry().Flow(flow.Name()).LossPct().Get(t); got > 0 {
			t.Fatalf("LossPct for flow %s: got %g, want 0", flow.Name(), got)
		}
	}

	for _, flow := range bad {
		if got := ate.Telemetry().Flow(flow.Name()).LossPct().Get(t); got < 100 {
			t.Fatalf("LossPct for flow %s: got %g, want 100", flow.Name(), got)
		}
	}
}

// awaitTimeout calls a fluent client Await, adding a timeout to the context.
func awaitTimeout(ctx context.Context, c *fluent.GRIBIClient, t testing.TB, timeout time.Duration) error {
	subctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Await(subctx, t)
}
