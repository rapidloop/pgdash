package api

import (
	"context"
	"math"
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
// Alerting-related definitions

// Filter predicates are used to filter out objects by name.
const (
	FilterPredNone = iota
	FilterPredContains
	FilterPredNotContains
	FilterPredStartsWith
	FilterPredEndsWith
	FilterPredEquals

	// these are the min and max values of FilterPred*
	_filterPredMin = FilterPredContains
	_filterPredMax = FilterPredEquals
)

// Filter type is the type of filter applied to the objects.
const (
	FilterNone = iota
	FilterPred
	FilterParent
)

// Filter optionally restricts an alert rule to a subset of objects.
type Filter struct {
	FilterType int    `json:"filter_type"`
	Predicate  int    `json:"pred"`
	Value      string `json:"value"`
}

// IsValid checks if the filter is valid.
func (f *Filter) IsValid() bool {
	switch f.FilterType {
	case FilterNone:
		// Predicate and Value are ignored
		return true
	case FilterPred:
		if f.Predicate < _filterPredMin || f.Predicate > _filterPredMax {
			return false
		}
		return len(f.Value) > 0 // Value is required
	case FilterParent:
		return len(f.Value) > 0 // Value is required, Predicate is ignored
	default:
		// invalid filter type
		return false
	}
}

// Units are the various units that may be associated with values in a rule.
const (
	UnitNone = iota
	UnitBytes
	UnitKiB
	UnitMiB
	UnitGiB
	UnitTiB
)

// Units for time.
const (
	UnitSeconds = iota + 101
	UnitMinutes
	UnitHours
	UnitDays
)

// Rule IDs
const (
	// server-level
	RuleServerWALCount          = 1  // WAL: number of files is g/t <count>
	RuleServerWALReadyCount     = 2  // WAL: number of files ready for archiving is g/t <count>
	RuleServerInactiveReplSlots = 3  // number of inactive repl slots is g/t <count>
	RuleServerPryWriteLag       = 4  // primary: write lag is g/t <bytes>
	RuleServerPryFlushLag       = 5  // primary: flush lag is g/t <bytes>
	RuleServerPryReplayLag      = 6  // primary: replay lag is g/t <bytes>
	RuleServerSbyReplayLagBytes = 7  // standby: replay lag is g/t <bytes>
	RuleServerSbyReplayLagTime  = 8  // standby: replay lag is g/t <time>
	RuleServerBELockWait        = 9  // no. of backends waiting for locks is g/t <count>
	RuleServerBEIdleInTxn       = 10 // no. of backends idling in txn is g/t <count>
	RuleServerBETxnOpen         = 11 // no. of backends with txn open for more than <time> is g/t <count>
	RuleServerTxnIDRange        = 12 // transaction id range is greater than <value> billion
	RuleServerLastCP            = 13 // time since last checkpoint is g/t <time>
	_ruleServerMin              = 1  // min of the RuleServer* values
	_ruleServerMax              = 13 // max of the RuleServer* values

	// database-level
	RuleDBBECount          = 101 // no. of backends is g/t <count>
	RuleDBBEPct            = 102 // no. of backends is g/t <pct> % of max
	RuleDBCommitRatio      = 103 // commit ratio is l/t <value>
	RuleDBTxnIDAge         = 104 // txn id age is g/t <pct> % of autvacuum_freeze_max_age
	RuleDBSize             = 105 // db size is g/t <bytes>
	RuleDBDisabledTriggers = 106 // disabled trigger count is g/t <count>
	RuleDBCacheHit         = 107 // cache hit ratio is l/t <pct> %
	_ruleDBMin             = 101 // min of the RuleDB* values
	_ruleDBMax             = 107 // max of the RuleDB* values

	// table-level
	RuleTableAutoVacTime = 201 // time since last auto vacuum is g/t <time>
	RuleTableAutoAnaTime = 202 // time since last auto analyze is g/t <time>
	RuleTableManVacTime  = 203 // time since last manual vacuum is g/t <time>
	RuleTableManAnaTime  = 204 // time since last manual analyze is g/t <time>
	RuleTableBloatBytes  = 205 // bloat is g/t <bytes>
	RuleTableBloatPct    = 206 // bloat is g/t <pct> % of table size
	RuleTableSize        = 207 // table size is g/t <bytes>
	RuleTableCacheHit    = 208 // cache hit ratio is l/t <pct> %
	_ruleTableMin        = 201 // min of the RuleTable* values
	_ruleTableMax        = 208 // max of the RuleTable* values

	// tablespace-level
	RuleTSSize          = 301 // table space size is g/t <bytes>
	RuleTSDiskFreePct   = 302 // free disk pct is l/t <pct> %
	RuleTSInodesFreePct = 303 // free inodes pct is l/t <pct> %
	_ruleTSMin          = 301 // min of the RuleTS* values
	_ruleTSMax          = 303 // max of the RuleTS* values
)

// rule traits: unit1 is none
func ruleNeedsUnit1None(r int) bool {
	return r == RuleServerWALCount || r == RuleServerWALReadyCount ||
		r == RuleServerInactiveReplSlots || r == RuleServerBELockWait ||
		r == RuleServerBEIdleInTxn || r == RuleServerTxnIDRange ||
		r == RuleDBBECount || r == RuleDBBEPct || r == RuleDBCommitRatio ||
		r == RuleDBTxnIDAge || r == RuleDBDisabledTriggers ||
		r == RuleDBCacheHit || r == RuleTableBloatPct || r == RuleTableCacheHit ||
		r == RuleTSDiskFreePct || r == RuleTSInodesFreePct
}

// rule traits: unit1 is bytes/k/m/g/b
func ruleNeedsUnit1Bytes(r int) bool {
	return r == RuleServerPryWriteLag || r == RuleServerPryFlushLag ||
		r == RuleServerPryReplayLag || r == RuleServerSbyReplayLagBytes ||
		r == RuleDBSize || r == RuleTableBloatBytes || r == RuleTableSize ||
		r == RuleTSSize
}

// rule traits: unit1 is sec/min/hr/day
func ruleNeedsUnit1Time(r int) bool {
	return r == RuleServerSbyReplayLagTime || r == RuleTableAutoVacTime ||
		r == RuleTableAutoAnaTime || r == RuleTableManVacTime ||
		r == RuleTableManAnaTime || r == RuleServerBETxnOpen ||
		r == RuleServerLastCP
}

// AlertingRule represents a single alerting rule.
type AlertingRule struct {
	RuleType int     `json:"rule_type"`
	Filter   *Filter `json:"filter,omitempty"`
	Value1   float64 `json:"value1"`
	Unit1    int     `json:"unit1,omitempty"`
	Value2   float64 `json:"value2,omitempty"`
	Unit2    int     `json:"unit2,omitempty"`
	Warn     bool    `json:"warn"`
}

// IsValid checks if this alerting rule is valid or not.
func (a *AlertingRule) IsValid() bool {
	// check if filter is appropriate for the rule (also rule type range)
	switch {
	case a.RuleType >= _ruleServerMin && a.RuleType <= _ruleServerMax:
		if a.Filter != nil && a.Filter.FilterType != FilterNone {
			// filter cannot be set at server-level
			return false
		}
	case (a.RuleType >= _ruleDBMin && a.RuleType <= _ruleDBMax) ||
		(a.RuleType >= _ruleTSMin && a.RuleType <= _ruleTSMax):
		if a.Filter != nil {
			// if filter is present, it must be valid
			if !a.Filter.IsValid() {
				return false
			}
			// only none & predicate filter allowed for db and tablespace
			if a.Filter.FilterType != FilterNone && a.Filter.FilterType != FilterPred {
				return false
			}
		}
	case a.RuleType >= _ruleTableMin && a.RuleType <= _ruleTableMax:
		if a.Filter != nil {
			// if filter is present, it must be valid
			if !a.Filter.IsValid() {
				return false
			}
		}
	default:
		// unknown rule type
		return false
	}
	// value1 should always be present, should not be NaN/Inf, and >= 0
	if a.Value1 < 0 || math.IsNaN(a.Value1) || math.IsInf(a.Value1, 0) {
		return false
	}
	// unit1 should be none for some rules, bytes/k/m/g/t for some rules and
	// sec/min/hr/day for some rules.
	switch a.Unit1 {
	case UnitNone:
		if !ruleNeedsUnit1None(a.RuleType) {
			return false // bad unit for rule
		}
	case UnitBytes, UnitKiB, UnitMiB, UnitGiB, UnitTiB:
		if !ruleNeedsUnit1Bytes(a.RuleType) {
			return false // bad unit for rule
		}
	case UnitSeconds, UnitMinutes, UnitHours, UnitDays:
		if !ruleNeedsUnit1Time(a.RuleType) {
			return false // bad unit for rule
		}
	default:
		return false // unknown unit
	}
	// only for rule RuleServerBETxnOpen, we have a second value
	if a.RuleType == RuleServerBETxnOpen {
		if a.Value2 < 0 || math.IsNaN(a.Value2) || math.IsInf(a.Value2, 0) {
			return false // value2 must be present
		}
		if a.Unit2 != UnitNone {
			return false // unit2 must be UnitNone
		}
	}
	return true
}

// AlertSettings represents the entire alert settings for a single server.
type AlertSettings struct {
	Version int            `json:"version"`
	Rules   []AlertingRule `json:"rules,omitempty"`
	Emails  []string       `json:"emails,omitempty"`
}

// The overall status of an alert.
const (
	StatusNoData   = -1
	StatusOK       = 0
	StatusBreached = 1
)

// AlertingRuleStatus is the status of one single AlertingRule.
type AlertingRuleStatus struct {
	Status     int     `json:"status"`           // one of Status*
	Database   string  `json:"db,omitempty"`     // database name, if any
	Table      string  `json:"table,omitempty"`  // schema-qualified table name, if any
	Tablespace string  `json:"tblspc,omitempty"` // tablespace name, if any
	Value      float64 `json:"value"`            // the actual value
	Units      string  `json:"units,omitempty"`  // the units for the value
}

// AlertStatus is the status of all rules, and corresponds to AlertSettings.
type AlertStatus struct {
	Version    int                    `json:"version"`
	RuleStatus [][]AlertingRuleStatus `json:"status,omitempty"`
}

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
	if ver != "1.0" && ver != "1.1" && ver != "1.2" {
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
	if ver != "1.0" && ver != "1.1" && ver != "1.2" {
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
