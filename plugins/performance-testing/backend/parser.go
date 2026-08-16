// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"regexp"
	"strconv"
)

// sysbench's plain-text report is the only output format its `run` command
// produces (no --output=json flag exists in the 1.0.x line), so the Job's
// log output has to be scraped with a regex rather than unmarshalled.
//
// The queries-per-second line's label genuinely differs across sysbench
// builds: a real run against pg-poc using sysbench 1.0.20 (Debian
// trixie's packaged build, see ../Dockerfile.sysbench) reported it as
// "queries:", not "queries performed:" as some other 1.0.x builds do
// (severalnines/sysbench, an earlier candidate image, used the "queries
// performed:" label — see job.go's sysbenchImage comment for why that
// image was dropped for an unrelated libpq reason). This was only caught
// by running an actual benchmark and inspecting the real output; queriesPerSecRE
// matches both labels.
//
//	transactions:                        3283   (328.09 per sec.)
//	queries:                             52528  (5249.37 per sec.)
//	...
//	    avg:                                    6.09
//	    95th percentile:                       54.83
var (
	transactionsPerSecRE = regexp.MustCompile(`(?m)^\s*transactions:\s+\d+\s+\(([\d.]+) per sec\.\)`)
	queriesPerSecRE      = regexp.MustCompile(`(?m)^\s*queries(?: performed)?:\s+\d+\s+\(([\d.]+) per sec\.\)`)
	latencyAvgRE         = regexp.MustCompile(`(?m)^\s*avg:\s+([\d.]+)\s*$`)
	latencyP95RE         = regexp.MustCompile(`(?m)^\s*95th percentile:\s+([\d.]+)\s*$`)
)

// parseSysbenchOutput extracts throughput/latency numbers from sysbench's
// stdout. Fields that don't match (unexpected sysbench version, truncated
// output) are simply left nil on the returned Result rather than treated
// as a parse error — a partial result is more useful than no result.
func parseSysbenchOutput(output string) *Result {
	result := &Result{}
	if m := transactionsPerSecRE.FindStringSubmatch(output); m != nil {
		result.TransactionsPerSec = parseFloatPtr(m[1])
	}
	if m := queriesPerSecRE.FindStringSubmatch(output); m != nil {
		result.QueriesPerSec = parseFloatPtr(m[1])
	}
	if m := latencyAvgRE.FindStringSubmatch(output); m != nil {
		result.LatencyAvgMs = parseFloatPtr(m[1])
	}
	if m := latencyP95RE.FindStringSubmatch(output); m != nil {
		result.LatencyP95Ms = parseFloatPtr(m[1])
	}
	return result
}

func parseFloatPtr(s string) *float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
