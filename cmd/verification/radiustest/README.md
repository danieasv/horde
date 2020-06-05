# RADIUS tester

This is a utility to test if the RADIUS interface in Horde is working and is
configured properly.

## What's tested

The client connects to the RADIUS server (set via the `--radius-endpoint` parameter), sends a request and verifies that the response matches the expectations

## Command line parameters

Attributes that aren't set will be omitted from the request.

```text
  -accept-expected
       Expect accept-response from RADIUS server (default true)
  -attr-imsi string
        IMSI attribute in RADIUS request (default "999912345678")
  -attr-ms-time-zone string
        MS Time Zone attribute in RADIUS request (default "@")
  -attr-nas-identifier string
        NAS identifier attribute in RADIUS request (default "NAS01")
  -attr-nas-ip string
        NAS IP Address attribute in RADIUS request (default "192.0.1.2")
  -attr-password string
        Password attribute in RADIUS request (default "password")
  -attr-user-location-info string
        User-location-info attribute in RADIUS request
  -attr-user-name string
        User name attribute in RADIUS request (default "4799887755")
  -expected-cidr string
        Expected CIDR range (default "127.0.0.1/8")
  -radius-endpoint string
        Host:port string for RADIUS server (default "127.0.0.1:1812")
  -shared-secret string
        Shared secret for the RADIUS server (default "secret")
```

## Return codes

The error code from the client indicates the types of error

| Return code | Description
| ----------- | -----------
| 0           | Successful
| 1           | Invalid command line parameter
| 2           | Invalid value in one of the command line parameter
| 3           | Connectivity error, either network or shared secret
| 4           | Server responded but output wasn't the expected
