package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cesanta/validate-json/schema"
)

var (
	schemaFile = flag.String("schema", "", "Path to schema to use.")
)

func main() {
	flag.Parse()

	if *schemaFile == "" {
		fmt.Fprintf(os.Stderr, "Need --schema\n")
		os.Exit(1)
	}

	b, err := ioutil.ReadFile(*schemaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read %q: %s\n", *schemaFile, err)
		os.Exit(1)
	}

	err = schema.ValidateDraft04Schema(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Schema is not valid: %s\n", err)
		os.Exit(1)
	}
}
