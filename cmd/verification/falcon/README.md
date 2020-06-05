# Falcon - Horde test tool

This is a tool to generate test data for Horde. There's some setup required
and you'll need `curl` or another HTTP client to generate some of the data.

To run it against a test server running locally do this:

1) Launch the Horde service with `make run` in the root directory. This will launch
a test server listening on the local loopback adapter.
2) Create a new API token by running `curl -XPOST -d'{"resource":"/","write":true}' localhost:8080/tokens``
3) Copy and paste the token into a new terminal and run `sudo bin/falcon --api-token=<token>`

The Falcon test command will run RADIUS requests agains the Horde instance, then
send a series of test messages. You will see output like this on the console:
```text
LOG     2019/06/12 23:24:19 radius.go:85: Sending radius request to 127.0.0.1:1812 for device 17dh0cf43jfgl9
LOG     2019/06/12 23:24:19 radius.go:99: IP address for device 17dh0cf43jfgl9 is 127.1.0.1
LOG     2019/06/12 23:24:20 deviceEmulator.go:41: Sending 12 bytes to 127.0.0.1:31415 from 127.1.0.1:12000
LOG     2019/06/12 23:24:20 outputs.go:65: Got 12 bytes from output: [0 0 0 0 21 167 144 124 110 22 167 40].
LOG     2019/06/12 23:24:20 outputs.go:36: RTT time for message from device 17dh0cf43jfgl9 with seq 0 is 1582 us
```

The roundtrip time for each message is printed for the webhook output while the UDP output will
just output the same 12 bytes sent.

The number of messages and devices can be set via the `--message-count` and `--device-count` parameters.
The `-message-interval` parameter controls the interval between messages **per device**, ie
100 devices will send once ever 1s so the net load is 100 messages/second.

Note: The test server is using SQLite and it does not behave wery well under load.

