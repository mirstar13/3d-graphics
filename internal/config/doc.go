// Package config loads and validates YAML renderer settings.
//
// Load starts with Default values and overlays one YAML document. Unknown
// fields, duplicate keys, invalid enum values, and out-of-range numbers are
// errors. A missing file is reported to the caller; the CLI decides whether
// falling back to defaults is appropriate.
//
// Settings.Validate must succeed before EngineConfig is used. Save validates
// before writing a temporary file in the destination directory, then renames
// it over the requested file. A failed validation leaves the existing file
// untouched. Saving emits canonical YAML and replaces existing comments.
package config
