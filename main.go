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

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rapidloop/pgdash/api"
	"github.com/rapidloop/pgmetrics"

	"github.com/pborman/getopt"
)

const usage = `pgdash is a command-line tool for talking to the pgDash application.

Usage:
  pgdash [OPTION]... COMMAND [ARGS]...

General options:
      --timeout=SECS       individual operation timeout in seconds (default: 60)
      --retries=COUNT      retry these many times on network or server errors (default: 5)
  -i, --input=FILE         read from this JSON file instead of stdin
  -a, --api-key=APIKEY     the API key for your pgDash account
      --base-url=URL       for use with self-hosted version of pgDash, see docs
  -V, --version            output version information, then exit
      --debug              output debugging information
  -h, --help[=options]     show this help, then exit
      --help=variables     list environment variables, then exit

Commands:
  report SERVERNAME        send report for PostgreSQL server SERVERNAME
  report-pgbouncer SERVERNAME PGBOUNCERNAME
                           send PgBouncer report for PgBouncer instance PGBOUNCERNAME
                               pooling connections for PostgreSQL server SERVERNAME

For more information, visit <https://pgdash.io>.
`

const variables = `Environment variables:
Usage:
  NAME=VALUE [NAME=VALUE] pgdash ...

  PDAPIKEY           API key for your pgdash account
`

var version string // set during build

var client *api.RestV1Client

const baseURL = "https://app.pgdash.io/api/v1"

type options struct {
	// general
	timeoutSec uint
	retries    uint
	input      string
	apiKey     string
	version    bool
	help       string
	helpShort  bool
	baseURL    string
	debug      bool
}

func (o *options) defaults() {
	// general
	o.timeoutSec = 60
	o.retries = 5
	o.input = ""
	o.apiKey = ""
	o.version = false
	o.help = ""
	o.helpShort = false
	o.baseURL = baseURL
	o.debug = false
}

func (o *options) usage(code int) {
	fp := os.Stdout
	if code != 0 {
		fp = os.Stderr
	}
	if o.helpShort || code != 0 || o.help == "short" {
		fmt.Fprint(fp, usage)
	} else if o.help == "variables" {
		fmt.Fprint(fp, variables)
	}
	os.Exit(code)
}

func printTry() {
	fmt.Fprint(os.Stderr, "Try \"pgdash --help\" for more information.\n")
}

func (o *options) parse() (args []string) {
	// make getopt
	s := getopt.New()
	s.SetUsage(printTry)
	s.SetProgram("pgdash")
	// general
	s.UintVarLong(&o.timeoutSec, "timeout", 0, "")
	s.UintVarLong(&o.retries, "retries", 0, "")
	s.StringVarLong(&o.input, "input", 'i', "")
	s.StringVarLong(&o.apiKey, "api-key", 'a', "")
	help := s.StringVarLong(&o.help, "help", 'h', "").SetOptional()
	s.BoolVarLong(&o.version, "version", 'V', "").SetFlag()
	s.StringVarLong(&o.baseURL, "base-url", 0, "")
	s.BoolVarLong(&o.debug, "debug", 0, "").SetFlag()

	// parse
	s.Parse(os.Args)
	if help.Seen() && o.help == "" {
		o.help = "short"
	}

	// check environment variables
	if o.apiKey == "" {
		if v := os.Getenv("PDAPIKEY"); v != "" {
			o.apiKey = v
		}
	}

	// check values
	if o.help != "" && o.help != "short" && o.help != "variables" {
		printTry()
		os.Exit(2)
	}
	if o.timeoutSec == 0 {
		fmt.Fprintln(os.Stderr, "timeout must be greater than 0")
		printTry()
		os.Exit(2)
	}
	if o.retries <= 1 {
		fmt.Fprintln(os.Stderr, "retries must be greater than 1")
		printTry()
		os.Exit(2)
	}

	// help action
	if o.helpShort || o.help == "short" || o.help == "variables" {
		o.usage(0)
	}

	// version action
	if o.version {
		if len(version) == 0 {
			version = "devel"
		}
		fmt.Println("pgdash", version)
		os.Exit(0)
	}

	// check the command
	args = s.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "a command must be specified")
		printTry()
		os.Exit(2)
	}
	command := args[0]
	if command != "report" && command != "report-pgbouncer" {
		fmt.Fprintf(os.Stderr, "unknown command '%s'\n", command)
		printTry()
		os.Exit(2)
	}

	return args
}

const sixMonths = time.Duration(180 * 24 * time.Hour)

func getReport(o options) *pgmetrics.Model {
	// read input file
	var data []byte
	var err error
	if len(o.input) > 0 {
		data, err = os.ReadFile(o.input)
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		log.Fatalf("failed to read input: %v", err)
	}
	if o.debug {
		log.Printf("read input: %d bytes", len(data))
	}

	// unmarshal json
	var model pgmetrics.Model
	if err := json.Unmarshal(data, &model); err != nil {
		log.Fatalf("invalid input: %v", err)
	}
	if o.debug {
		log.Print("decoded input JSON successfully")
	}

	// validate the data a bit
	ver := model.Metadata.Version
	if !strings.HasPrefix(ver, "1.") { // we currently know only about major version 1
		log.Fatalf("invalid input: bad schema version '%s' in pgmetrics json",
			ver)
	}
	at := time.Unix(model.Metadata.At, 0)
	now := time.Now()
	if at.Before(now.Add(-sixMonths)) || at.After(now.Add(sixMonths)) {
		log.Fatalf("invalid input: bad collection timestamp in pgmetrics json: %v", at)
	}

	// append our user agent info into the model
	if len(model.Metadata.UserAgent) > 0 {
		model.Metadata.UserAgent += " "
	}
	model.Metadata.UserAgent += "pgdash/"
	if len(version) > 0 {
		model.Metadata.UserAgent += version
	} else {
		model.Metadata.UserAgent += "devel"
	}

	return &model
}

func checkAPIKey(o options) {
	if len(o.apiKey) == 0 {
		log.Fatal("API key must be specified using the '-a' option for reporting.")
	}
	if !api.RxAPIKey.MatchString(o.apiKey) {
		log.Fatalf("invalid API key format '%s'", o.apiKey)
	}
}

func cmdReport(o options, args []string) {
	// check API key
	checkAPIKey(o)

	// check server
	if len(args) == 0 {
		log.Fatal("Server name needs to be specified, try --help for help.")
	}
	if len(args) != 1 {
		log.Fatal("invalid syntax for report command, try --help for help.")
	}
	if !api.RxServer.MatchString(args[0]) {
		log.Fatal(`bad server name, must be 1-64 chars A-Z, a-z, 0-9, "-", "_", and ".".`)
	}

	// check the model (must not have pgbouncer info)
	model := getReport(o)
	if model.PgBouncer != nil {
		log.Fatal("use report-pgbouncer to send PgBouncer information")
	}

	// call the api
	_, err := client.Report(api.ReqReport{
		APIKey: o.apiKey,
		Server: args[0],
		Data:   *model,
	})
	if errh, ok := err.(*api.RestV1ClientError); ok {
		if errh.Code() == 400 {
			log.Fatal("invalid API key or account limit reached")
		}
		if errh.Code() == 500 {
			log.Fatal("internal server error")
		}
	}
	if err != nil {
		log.Fatalf("API request failed: %v", err)
	}
}

func cmdReportPgBouncer(o options, args []string) {
	// check API key
	checkAPIKey(o)

	// check args
	if len(args) != 2 {
		log.Fatal("invalid syntax for report-pgbouncer command, try --help for help.")
	}
	if !api.RxServer.MatchString(args[0]) {
		log.Fatal(`bad server name, must be 1-64 chars A-Z, a-z, 0-9, "-", "_", and ".".`)
	}
	if !api.RxServer.MatchString(args[1]) {
		log.Fatal(`bad PgBouncer name, must be 1-64 chars A-Z, a-z, 0-9, "-", "_", and ".".`)
	}

	// check the model (must have pgbouncer info)
	model := getReport(o)
	if model == nil || model.PgBouncer == nil {
		log.Fatal("pgmetrics report does not contain PgBouncer information")
	}

	// call the api
	_, err := client.ReportPgBouncer(api.ReqReportPgBouncer{
		APIKey:    o.apiKey,
		Server:    args[0],
		PgBouncer: args[1],
		Data:      *model,
	})
	if errh, ok := err.(*api.RestV1ClientError); ok {
		if errh.Code() == 400 {
			log.Fatalf("invalid API key or server %q not found", args[0])
		}
		if errh.Code() == 500 {
			log.Fatal("internal server error")
		}
	}
	if err != nil {
		log.Fatalf("API request failed: %v", err)
	}
}

func main() {
	var o options
	o.defaults()
	args := o.parse()
	command := args[0]

	log.SetPrefix("pgdash: ")
	if o.debug {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	} else {
		log.SetFlags(0)
	}

	// create the client
	tout := time.Duration(o.timeoutSec) * time.Second
	client = api.NewRestV1Client(o.baseURL, tout, int(o.retries))
	client.SetDebug(o.debug)

	switch command {
	case "report":
		cmdReport(o, args[1:])
	case "report-pgbouncer":
		cmdReportPgBouncer(o, args[1:])
	}
}
