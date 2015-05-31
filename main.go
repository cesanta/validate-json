// validate-json is a utility for validation JSON values with JSON Schema.
//
// Usage example:
//
//   validate-json --schema path/to/schema.json --input path/to/data.json
//
// If everything is fine it will exit with status 0 and without any output. If
// there was any errors, exit code will be non-zero and errors will be printed
// to stderr.
//
// Additional flags:
//
//   --extra "schema1.json schema2.json ..."
// Space-separated list of additional schema files to load so they can be
// referenced from the primary schema. Each of the schemas in these files needs
// to have "id" set.
//
//   -n
// If present, referenced schemas will be fetched from the remote hosts.
package main

import (
	"flag"
	"fmt"
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

	f, err := os.Open(*schemaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %q: %s\n", *schemaFile, err)
		os.Exit(1)
	}

	s, err := schema.ParseDraft04Schema(f)
	f.Close()
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

	f, err = os.Open(*inputFile)
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
