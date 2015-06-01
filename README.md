# JSON Schema validator

[![GoDoc](https://godoc.org/github.com/cesanta/validate-json/schema?status.svg)](https://godoc.org/github.com/cesanta/validate-json/schema)

This binary is a command-line wrapper for a library that implements [JSON Schema
draft 04 specification](http://json-schema.org/documentation.html).
It passes all the tests from https://github.com/json-schema/JSON-Schema-Test-Suite
except for optional/bignum.json, but it doesn't mean that it's free of bugs,
especially in scope alteration and resolution, since that part is not entrirely
clear.
