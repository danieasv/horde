# ctrlh - the horde control CLI

This is a CLI to manage and set up Horde.

## Usage

```text
Usage: ctrlh --endpoint="localhost:1234" <command>

Horde management CLI

Flags:
  --help                         Show context-sensitive help.
  --endpoint="localhost:1234"    gRPC management endpoint
  --tls                          TLS enabled for gRPC
  --cert-file=STRING             Client certificate for management service
  --hostname-override=STRING     Host name override for certificate

Commands:
  ping          Ping the management service
  apn add       Add new APN in Horde
  apn rm        Remove APN from Horde
  apn list      List APNs registered in Horde
  nas add       Add new NAS
  nas rm        Remove existing NAS
  nas list      List NASes
  alloc add     Add allocation for device
  alloc rm      Remove allocation for device
  alloc list    List address allocations
  token add     Create a new API token for an user
  token rm      Remove an API token from an existing user
  user add      Create new API user and associated token in Horde
  util id       Decode API identifiers to internal identifiers
  util di       Encode internal identifiers into API identifiers

Run "ctrlh <command> --help" for more information on a command.
```
