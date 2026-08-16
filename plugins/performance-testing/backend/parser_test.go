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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realSysbenchOutput is a real sysbench 1.0.x oltp_read_write report,
// trimmed to the sections parseSysbenchOutput actually reads.
const realSysbenchOutput = `sysbench 1.0.20 (using bundled LuaJIT 2.1.0-beta2)

Running the test with following options:
Number of threads: 8

SQL statistics:
    queries performed:
        read:                            140060
        write:                           40018
        other:                           20009
        total:                           200087
    transactions:                        10004  (1000.31 per sec.)
    queries performed:                   200087 (20005.85 per sec.)
    ignored errors:                      0      (0.00 per sec.)
    reconnects:                          0      (0.00 per sec.)

General statistics:
    total time:                          10.0007s
    total number of events:              10004

Latency (ms):
         min:                                    1.34
         avg:                                    1.60
         max:                                   15.29
         95th percentile:                        2.13
         sum:                                 15987.75
`

func TestParseSysbenchOutput_RealReport(t *testing.T) {
	t.Parallel()

	result := parseSysbenchOutput(realSysbenchOutput)

	require.NotNil(t, result.TransactionsPerSec)
	assert.InDelta(t, 1000.31, *result.TransactionsPerSec, 0.001)

	require.NotNil(t, result.QueriesPerSec)
	assert.InDelta(t, 20005.85, *result.QueriesPerSec, 0.001)

	require.NotNil(t, result.LatencyAvgMs)
	assert.InDelta(t, 1.60, *result.LatencyAvgMs, 0.001)

	require.NotNil(t, result.LatencyP95Ms)
	assert.InDelta(t, 2.13, *result.LatencyP95Ms, 0.001)
}

// actualDebianTrixieSysbenchOutput is the real, unedited output of a
// sysbench 1.0.20 (Debian trixie's packaged build, ../Dockerfile.sysbench)
// oltp_read_only smoke run against the live pg-poc PostgreSQL instance on
// everest-poc. Unlike the older sysbench build realSysbenchOutput above
// was modelled on, this build's per-second summary line reads "queries:",
// not "queries performed:" — queriesPerSecRE originally only matched the
// latter and silently dropped QueriesPerSec on every real run until this
// was caught by actually running a benchmark and checking GET
// /api/runs/{id}'s response, not by inspecting the code.
const actualDebianTrixieSysbenchOutput = `sysbench 1.0.20 (using system LuaJIT 2.1.1700206165)

Running the test with following options:
Number of threads: 2
Initializing random number generator from current time


Initializing worker threads...

Threads started!

SQL statistics:
    queries performed:
        read:                            45962
        write:                           0
        other:                           6566
        total:                           52528
    transactions:                        3283   (328.09 per sec.)
    queries:                             52528  (5249.37 per sec.)
    ignored errors:                      0      (0.00 per sec.)
    reconnects:                          0      (0.00 per sec.)

General statistics:
    total time:                          10.0035s
    total number of events:              3283

Latency (ms):
         min:                                    0.85
         avg:                                    6.09
         max:                                   69.14
         95th percentile:                       54.83
         sum:                                19992.04
`

func TestParseSysbenchOutput_ActualDebianTrixieBuild_QueriesLineHasNoPerformedSuffix(t *testing.T) {
	t.Parallel()

	result := parseSysbenchOutput(actualDebianTrixieSysbenchOutput)

	require.NotNil(t, result.TransactionsPerSec)
	assert.InDelta(t, 328.09, *result.TransactionsPerSec, 0.001)

	require.NotNil(t, result.QueriesPerSec, "queries-per-sec must parse even when the line reads \"queries:\" instead of \"queries performed:\"")
	assert.InDelta(t, 5249.37, *result.QueriesPerSec, 0.001)

	require.NotNil(t, result.LatencyAvgMs)
	assert.InDelta(t, 6.09, *result.LatencyAvgMs, 0.001)

	require.NotNil(t, result.LatencyP95Ms)
	assert.InDelta(t, 54.83, *result.LatencyP95Ms, 0.001)
}

func TestParseSysbenchOutput_UnexpectedFormat_ReturnsPartialResult(t *testing.T) {
	t.Parallel()

	result := parseSysbenchOutput("sysbench: error: no such command\n")

	assert.Nil(t, result.TransactionsPerSec)
	assert.Nil(t, result.QueriesPerSec)
	assert.Nil(t, result.LatencyAvgMs)
	assert.Nil(t, result.LatencyP95Ms)
}
