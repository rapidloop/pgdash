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
