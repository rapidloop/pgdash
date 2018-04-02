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
)

//------------------------------------------------------------------------------

// RestV1 is the interface definition of the public REST API, v1.
type RestV1 interface {
	Quick(ctx context.Context, req ReqQuick) (resp RespQuick, code int)
}

//------------------------------------------------------------------------------
// RestV1.Quick

// ReqQuick is the request structure for RestV1.Quick.
type ReqQuick struct {
	Data pgmetrics.Model `json:"data"`
}

// IsValid returns true if this the fields in this object look valid.
func (r *ReqQuick) IsValid() bool {
	if r.Data.Metadata.Version != "1.0" {
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
