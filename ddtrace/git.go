// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package ddtrace

import (
	"gopkg.in/DataDog/dd-trace-go.v1/internal"
)

func ReportUnexportedGitInfo() (string, string) {
	return internal.ReportUnexportedGitInfo()
}