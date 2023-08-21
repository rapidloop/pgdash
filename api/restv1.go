/*
 * Copyright 2023 RapidLoop, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"context"
	"regexp"

	"github.com/rapidloop/pgmetrics"
)

var (
	// RxAPIKey is the regexp a valid API key should match.
	RxAPIKey = regexp.MustCompile("^[A-Za-z0-9]{22}$")

	// RxServer is the regexp a valid server name should match.
	RxServer = regexp.MustCompile("^[A-Za-z0-9_.-]{1,64}$")
)

//------------------------------------------------------------------------------

// RestV1 is the interface definition of the public REST API, v1.
type RestV1 interface {
	Report(ctx context.Context, req ReqReport) (resp RespReport, code int)
}

//------------------------------------------------------------------------------
// RestV1.Report

// ReqReport is the request structure for RestV1.Report.
type ReqReport struct {
	APIKey string          `json:"apikey"`
	Server string          `json:"server"`
	Data   pgmetrics.Model `json:"data"`
}

// RespReport is the response structure for RestV1.Report.
type RespReport struct {
}

//------------------------------------------------------------------------------
// RestV1.ReportPgBouncer

// ReqReportPgBouncer is the request structure for RestV1.ReportPgBouncer.
type ReqReportPgBouncer struct {
	APIKey    string          `json:"apikey"`
	Server    string          `json:"server"`
	PgBouncer string          `json:"pgbouncer"`
	Data      pgmetrics.Model `json:"data"`
}

//------------------------------------------------------------------------------
// RestV1.ReportPgpool

// ReqReportPgpool is the request structure for RestV1.ReportPgpool.
type ReqReportPgpool struct {
	APIKey string          `json:"apikey"`
	Pgpool string          `json:"pgpool"`
	Data   pgmetrics.Model `json:"data"`
}
