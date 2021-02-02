// Copyright 2021 The gVisor Authors.
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

// Package mitigate provides libraries for the mitigate command. The
// mitigate command mitigates side channel attacks such as MDS. Mitigate
// shuts down CPUs via /sys/devices/system/cpu/cpu{N}/online. In addition,
// the mitigate also handles computing available CPU in kubernetes kube_config
// files.
package mitigate

import (
	"fmt"

	"gvisor.dev/gvisor/pkg/log"
	"gvisor.dev/gvisor/runsc/flag"
)

// UsageString is the usage string for the mitigate command.
var UsageString = `mitigate [flags]

This command mitigates an underlying system against side channel attacks.
The command checks /proc/cpuinfo for cpus having key vulnerablilities (meltdown,
l1tf, mds, swapgs, taa). If cpus are found to have one of the vulnerabilities,
all but one cpu is shutdown on each core via
/sys/devices/system/cpu/cpu{N}/online.
`

// Mitigate handles high level mitigate operations provided to runsc.
type Mitigate struct {
	dryRun bool     // Run the command without changing the underlying system.
	other  mitigate // Struct holds extra mitigate logic.
}

// Usage implments Usage for cmd.Mitigate.
func (m Mitigate) Usage() string {
	return UsageString + usageString
}

// SetFlags sets flags for the command Mitigate.
func (m Mitigate) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&m.dryRun, "dryrun", false, "run the command without changing system")
	mitigate{}.setFlags(f)
}

// Execute executes the Mitigate command.
func (m Mitigate) Execute(data []byte) error {
	set, err := newCPUSet(data, m.other.vulnerable)
	if err != nil {
		return err
	}

	log.Infof("Mitigate found the following CPUs...")
	log.Infof("%s", set)

	shutdownList := set.getShutdownList()
	log.Infof("Shutting down threads on thread pairs.\n")
	for _, t := range shutdownList {
		log.Infof("Shutting down thread: %s", t)
		if m.dryRun {
			continue
		}
		if err := t.shutdown(); err != nil {
			return fmt.Errorf("error shutting down thread: %v\n %s", err, t)
		}
	}
	log.Infof("Shutdown successful.")
	m.other.execute(set, m.dryRun)
	return nil
}
