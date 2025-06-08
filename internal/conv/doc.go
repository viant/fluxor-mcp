// Package conv provides small, reflection-based helpers to convert between
// arbitrary Go values.  The primary helper Convert performs a best-effort JSON
// marshal/unmarshal round-trip which is sufficient for coercing data between
// maps, structs and primitive types when handling tool arguments and results.
package conv
