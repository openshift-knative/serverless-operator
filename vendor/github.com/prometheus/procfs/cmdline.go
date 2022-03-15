// Copyright 2021 The Prometheus Authors
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

<<<<<<<< HEAD:vendor/github.com/prometheus/procfs/cmdline.go
package procfs
========
//go:build !go1.15
// +build !go1.15

package collectors
>>>>>>>> 081960ee5 (Tests for EUS-to-EUS OpenShift upgrades):vendor/github.com/prometheus/client_golang/prometheus/collectors/dbstats_collector_pre_go115.go

import (
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

// CmdLine returns the command line of the kernel.
func (fs FS) CmdLine() ([]string, error) {
	data, err := util.ReadFileNoStat(fs.proc.Path("cmdline"))
	if err != nil {
		return nil, err
	}

	return strings.Fields(string(data)), nil
}
