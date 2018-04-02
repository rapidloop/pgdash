/*
 * Copyright 2018 RapidLoop, Inc.
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
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/rapidloop/pgdash/api"
	"github.com/rapidloop/pgmetrics"

	"github.com/pborman/getopt"
)

const usage = `pgdash is a command-line tool for talking to https://pgdash.io/.

Usage:
  pgdash [OPTION]... COMMAND [ARGS]...

General options:
      --timeout=SECS       individual operation timeout in seconds (default: 60)
      --retries=COUNT      retry these many times on network or server errors (default: 5)
  -i, --input=FILE         the input JSON file for "quick" command
  -a, --api-key=APIKEY     the API key for your pgdash account
  -V, --version            output version information, then exit
  -?, --help[=options]     show this help, then exit
      --help=variables     list environment variables, then exit

Commands:
  quick                    send the JSON output of pgmetrics from stdin to pgdash.io

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
}

func (o *options) defaults() {
	// general
	o.timeoutSec = 60
	o.retries = 5
}

func (o *options) usage(code int) {
	fp := os.Stdout
	if code != 0 {
		fp = os.Stderr
	}
	if o.helpShort || code != 0 || o.help == "short" {
		fmt.Fprintf(fp, usage)
	} else if o.help == "variables" {
		fmt.Fprint(fp, variables)
	}
	os.Exit(code)
}

func printTry() {
	fmt.Fprintf(os.Stderr, "Try \"pgdash --help\" for more information.\n")
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
	help := s.StringVarLong(&o.help, "help", '?', "").SetOptional()
	s.BoolVarLong(&o.version, "version", 'V', "").SetFlag()

	// parse
	s.Parse(os.Args)
	if help.Seen() && o.help == "" {
		o.help = "short"
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
	if command != "quick" {
		fmt.Fprintf(os.Stderr, "unknown command '%s'\n", command)
		printTry()
		os.Exit(2)
	}

	return args
}

func cmdQuick(o options, args []string) {
	// read input file
	var data []byte
	var err error
	if len(o.input) > 0 {
		data, err = ioutil.ReadFile(o.input)
	} else {
		data, err = ioutil.ReadAll(os.Stdin)
	}
	if err != nil {
		log.Fatalf("failed to read input: %v", err)
	}

	// unmarshal json
	var model pgmetrics.Model
	if err := json.Unmarshal(data, &model); err != nil {
		log.Fatalf("invalid input: %v", err)
	}

	// validate the data a bit
	if model.Metadata.Version != "1.0" { // we currently know only about this version
		log.Fatalf("invalid input: bad schema version '%s' in pgmetrics json",
			model.Metadata.Version)
	}
	at := time.Unix(model.Metadata.At, 0)
	if at.Year() < 2018 || at.Year() > 2020 {
		log.Fatal("invalid input: bad collection timestamp in pgmetrics json")
	}

	// call the api
	resp, err := client.Quick(api.ReqQuick{
		Data: model,
	})
	if err != nil {
		log.Fatalf("API request failed: %v", err)
	}

	// print out the result
	fmt.Printf(`Upload successful.

Quick View URL: %s
Admin Code:     %s
`, resp.URL, resp.Code)
}

func main() {
	var o options
	o.defaults()
	args := o.parse()
	command := args[0]

	log.SetFlags(0)
	log.SetPrefix("pgdash: ")

	// create the client
	tout := time.Duration(o.timeoutSec) * time.Second
	client = api.NewRestV1Client(baseURL, tout, int(o.retries))

	switch command {
	case "quick":
		cmdQuick(o, args[1:])
	}
}
