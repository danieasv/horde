module github.com/eesrc/horde

go 1.14

require (
	github.com/ExploratoryEngineering/logging v0.0.0-20181106085733-dcb8702a004e
	github.com/ExploratoryEngineering/params v1.0.0
	github.com/ExploratoryEngineering/pubsub v0.0.0-20190305221504-4b456ceeb4ae
	github.com/ExploratoryEngineering/rest v0.0.0-20181001125504-5b79b712352a
	github.com/TelenorDigital/goconnect v0.7.0
	github.com/alecthomas/kong v0.2.9
	github.com/davecgh/go-spew v1.1.1
	github.com/dustin/go-coap v0.0.0-20190908170653-752e0f79981e
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/go-ocf/go-coap v0.0.0-20200511140640-db6048acfdd3
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.6.0
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/nsf/termbox-go v0.0.0-20200418040025-38ba6e5628f1 // indirect
	github.com/prometheus/client_golang v1.6.0
	github.com/stretchr/testify v1.6.0
	github.com/telenordigital/nbiot-go v0.0.0-20200302150853-aec59ff03970
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200604104852-0b0486081ffb
	google.golang.org/grpc v1.29.1
	layeh.com/radius v0.0.0-20190322222518-890bc1058917
)

//replace github.com/go-ocf/go-coap v0.0.0-20191205091034-1fba24d18397 => ../go-ocf-local
