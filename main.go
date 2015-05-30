package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	json "github.com/cesanta/ucl"
	"github.com/cesanta/validate-json/schema"
)

var (
	schemaFile = flag.String("schema", "", "Path to schema to use.")
	inputFile  = flag.String("input", "", "Path to the JSON data to validate.")
	network    = flag.Bool("n", false, "If true, fetching of referred schemas from remote hosts will be enabled.")
	extra      = flag.String("extra", "", "Space-separated list of schema files to pre-load for the purpose of remote references. Each schema needs to have 'id' property.")
)

func main() {
	flag.Parse()

	if *schemaFile == "" || *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Need --schema and --input\n")
		os.Exit(1)
	}

	b, err := ioutil.ReadFile(*schemaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read %q: %s\n", *schemaFile, err)
		os.Exit(1)
	}

	s, err := schema.ParseDraft04Schema(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Schema is not valid: %s\n", err)
		os.Exit(1)
	}

	loader := schema.NewLoader()
	loader.EnableNetworkAccess(*network)
	if *extra != "" {
		for _, file := range strings.Split(*extra, " ") {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open %q: %s", file, err)
				os.Exit(1)
			}
			s, err := json.Parse(f)
			f.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse %q: %s", file, err)
				os.Exit(1)
			}
			loader.Add(s)
		}
	}

	validator := schema.NewValidator(s, loader)

	f, err := os.Open(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open input file: %s", err)
		os.Exit(1)
	}
	defer f.Close()
	data, err := json.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse input file: %s", err)
		os.Exit(1)
	}
	if err := validator.Validate(data); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
