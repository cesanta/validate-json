// Package schema implements JSON Schema draft 04 specification
// (http://json-schema.org/documentation.html).
// It passes all the tests from https://github.com/json-schema/JSON-Schema-Test-Suite
// except for optional/bignum.json, but it doesn't mean that it's free of bugs,
// especially in scope alteration and resolution, since that part is not entrirely
// clear.
//
// Usage example:
//
//   // Load the schema.
//   s, err := schema.ParseDraft04Schema(f)
//   if err != nil {
//      log.Fatalf("Schema is not valid: %s", err)
//   }
//
//   // Construct validator.
//   loader := schema.NewLoader()
//   loader.EnableNetworkAccess(*network)
//   validator := schema.NewValidator(s, loader)
//
//   // Validate some data.
//   if err := validator.Validate(data); err != nil {
//      log.Fatalf("Validation failed: %s", err)
//   }
package schema
