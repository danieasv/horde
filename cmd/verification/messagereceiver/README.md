# Message receiver

This is a tool that acts as a device receiving downstream messages. The
current message count is exposed through a HTTP service `/coap-pull`,
`/coap-push` and `/udp`. Every message is acked via the regular UDP interface.

