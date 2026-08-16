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

import "time"

// Workload names a predefined sysbench profile. Hiding raw sysbench flags
// behind a small, named set keeps the API surface simple for a first
// engine; a "custom" profile carrying raw parameters is future work, not
// this PoC.
type Workload string

const (
	WorkloadSmoke      Workload = "smoke"       // short, low-concurrency sanity check
	WorkloadReadHeavy  Workload = "read-heavy"  // oltp_read_only
	WorkloadWriteHeavy Workload = "write-heavy" // oltp_write_only
	WorkloadMixedOLTP  Workload = "mixed-oltp"  // oltp_read_write
)

// RunStatus is the lifecycle state of a benchmark run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "Pending"
	RunStatusRunning   RunStatus = "Running"
	RunStatusSucceeded RunStatus = "Succeeded"
	RunStatusFailed    RunStatus = "Failed"
)

// Result holds the throughput/latency numbers parsed out of sysbench's
// output. Pointers so an in-progress or failed run can omit fields that
// were never produced, rather than reporting a misleading zero.
type Result struct {
	TransactionsPerSec *float64 `json:"transactionsPerSec,omitempty"`
	QueriesPerSec      *float64 `json:"queriesPerSec,omitempty"`
	LatencyAvgMs       *float64 `json:"latencyAvgMs,omitempty"`
	LatencyP95Ms       *float64 `json:"latencyP95Ms,omitempty"`
}

// Run is one benchmark execution, from request through result. This is the
// unit the run Store persists and the list/get API returns.
type Run struct {
	ID         string     `json:"id"`
	Instance   string     `json:"instance"`
	Namespace  string     `json:"namespace"`
	Workload   Workload   `json:"workload"`
	Status     RunStatus  `json:"status"`
	JobName    string     `json:"jobName"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Result     *Result    `json:"result,omitempty"`
	Error      string     `json:"error,omitempty"`
}
