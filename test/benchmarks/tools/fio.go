// Copyright 2020 The gVisor Authors.
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

package tools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// Fio makes 'fio' commands and parses their output.
type Fio struct {
	Test      string // test to run: read, write, randread, randwrite.
	Size      int    // total size to be read/written in megabytes.
	BlockSize int    // block size to be read/written in kilobytes.
	IODepth   int    // I/O depth for reads/writes.
	Direct    bool   // Whether to use direct I/O (O_DIRECT) or not.
}

// MakeCmd makes a 'fio' command.
func (f *Fio) MakeCmd(filename string) []string {
	cmd := []string{"fio", "--output-format=json", "--ioengine=sync"}
	cmd = append(cmd, fmt.Sprintf("--name=%s", f.Test))
	cmd = append(cmd, fmt.Sprintf("--size=%dM", f.Size))
	cmd = append(cmd, fmt.Sprintf("--blocksize=%dK", f.BlockSize))
	cmd = append(cmd, fmt.Sprintf("--filename=%s", filename))
	cmd = append(cmd, fmt.Sprintf("--iodepth=%d", f.IODepth))
	if f.Direct {
		cmd = append(cmd, "--direct=1")
	} else {
		cmd = append(cmd, "--direct=0")
	}
	cmd = append(cmd, fmt.Sprintf("--rw=%s", f.Test))
	if f.Test == "read" || f.Test == "randread" {
		// Don't call `fallocate` during read-only tests.
		// Calling `fallocate` is not a typical operation for an application to do
		// when it is only trying to read a file.
		// This has performance implications for gVisor, so we override fio's
		// default behavior for read-only benchmarks to be more representative of
		// real-world read-only performance.
		cmd = append(cmd, "--fallocate=none")
	}
	return cmd
}

// Report reports metrics based on output from an 'fio' command.
func (f *Fio) Report(b *testing.B, output string) {
	b.Helper()
	// Parse the output and report the metrics.
	isRead := strings.Contains(f.Test, "read")
	bw, err := f.parseBandwidth(output, isRead)
	if err != nil {
		b.Fatalf("failed to parse bandwidth from %s with: %v", output, err)
	}
	ReportCustomMetric(b, bw, "bandwidth" /*metric name*/, "bytes_per_second" /*unit*/)

	iops, err := f.parseIOps(output, isRead)
	if err != nil {
		b.Fatalf("failed to parse iops from %s with: %v", output, err)
	}
	ReportCustomMetric(b, iops, "io_ops" /*metric name*/, "ops_per_second" /*unit*/)
}

// parseBandwidth reports the bandwidth in b/s.
func (f *Fio) parseBandwidth(data string, isRead bool) (float64, error) {
	op := "write"
	if isRead {
		op = "read"
	}
	result, err := f.parseFioJSON(data, op, "bw")
	if err != nil {
		return 0, err
	}
	return result * 1024, nil
}

// parseIOps reports the write IO per second metric.
func (f *Fio) parseIOps(data string, isRead bool) (float64, error) {
	if isRead {
		return f.parseFioJSON(data, "read", "iops")
	}
	return f.parseFioJSON(data, "write", "iops")
}

// fioResult is for parsing FioJSON.
type fioResult struct {
	Jobs []fioJob
}

// fioJob is for parsing FioJSON.
type fioJob map[string]json.RawMessage

// fioMetrics is for parsing FioJSON.
type fioMetrics map[string]json.RawMessage

// parseFioJSON parses data and grabs "op" (read or write) and "metric"
// (bw or iops) from the JSON.
func (f *Fio) parseFioJSON(data, op, metric string) (float64, error) {
	var result fioResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return 0, fmt.Errorf("could not unmarshal data: %v", err)
	}

	if len(result.Jobs) < 1 {
		return 0, fmt.Errorf("no jobs present to parse")
	}

	var metrics fioMetrics
	if err := json.Unmarshal(result.Jobs[0][op], &metrics); err != nil {
		return 0, fmt.Errorf("could not unmarshal jobs: %v", err)
	}

	if _, ok := metrics[metric]; !ok {
		return 0, fmt.Errorf("no metric found for op: %s", op)
	}
	return strconv.ParseFloat(string(metrics[metric]), 64)
}
