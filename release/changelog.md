# v{{.Version}}: {{.Name}}

## API

Received field on upstream messages is now represented as an int64 encoded
in a string. Upgrade nbiot-go client to 0.6.0 to fix this. JavaScript and
most other dynamically typed languages will handle this fine.

Commit hash: {{.CommitHash}}
