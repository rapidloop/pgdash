package api

import (
	"context"
	"regexp"
	"time"

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
	Quick(ctx context.Context, req ReqQuick) (resp RespQuick, code int)
	Report(ctx context.Context, req ReqReport) (resp RespReport, code int)
}

//------------------------------------------------------------------------------
// RestV1.Quick

// ReqQuick is the request structure for RestV1.Quick.
type ReqQuick struct {
	Data pgmetrics.Model `json:"data"`
}

// IsValid returns true if this the fields in this object look valid.
func (r *ReqQuick) IsValid() bool {
	ver := r.Data.Metadata.Version
	if ver != "1.0" && ver != "1.1" {
		return false
	}
	at := time.Unix(r.Data.Metadata.At, 0)
	if at.Year() < 2018 || at.Year() > 2020 {
		return false
	}
	return true
}

// RespQuick is the response structure for RestV1.Quick.
type RespQuick struct {
	URL  string `json:"url"`
	Code string `json:"code"`
}

//------------------------------------------------------------------------------
// RestV1.Report

// ReqReport is the request structure for RestV1.Report.
type ReqReport struct {
	APIKey string          `json:"apikey"`
	Server string          `json:"server"`
	Data   pgmetrics.Model `json:"data"`
}

// IsValid returns true if this the fields in this object look valid.
func (r *ReqReport) IsValid() bool {
	if !RxAPIKey.MatchString(r.APIKey) {
		return false
	}
	if !RxServer.MatchString(r.Server) {
		return false
	}
	ver := r.Data.Metadata.Version
	if ver != "1.0" && ver != "1.1" {
		return false
	}
	at := time.Unix(r.Data.Metadata.At, 0)
	if at.Year() < 2018 || at.Year() > 2020 {
		return false
	}
	return true
}

// RespReport is the response structure for RestV1.Report.
type RespReport struct {
}
