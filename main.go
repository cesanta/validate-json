package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	json "github.com/cesanta/ucl"
	"github.com/cesanta/validate-json/schema"
)

var (
	schemaFile = flag.String("schema", "", "Path to schema to use.")
	inputFile  = flag.String("input", "", "Path to the JSON data to validate.")
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

	validator := schema.NewValidator(s, nil)

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
