#if GOPATH isn't set, point it to the default
ifeq ($(GOPATH),)
GOPATH := $(HOME)/go
endif

version := $(shell reto version)
commit := $(shell reto hash)
name := $(shell reto hashname)
build_time := $(shell date +"%Y-%m-%dT%H:%M")
ldflags := -X github.com/eesrc/horde/pkg/version.Number=$(version) \
	-X github.com/eesrc/horde/pkg/version.CommitHash=$(commit) \
	-X github.com/eesrc/horde/pkg/version.Name=$(name) \
	-X github.com/eesrc/horde/pkg/version.BuildDate=$(build_time)

all: test lint vet build

clean:
	find . -name "*-wal" -delete
	find . -name "*-shm" -delete
	rm -f bin/*.linux

build: horde magpie ctrlh verification

horde:
	cd cmd/horde &&	go build -o ../../bin/horde
	cd cmd/ingress/horde-udp && go build -o ../../../bin/horde-udp
	cd cmd/ingress/horde-coap && go build -o ../../../bin/horde-coap
	cd cmd/ingress/horde-radius && go build -o ../../../bin/horde-radius

magpie:
	cd cmd/magpie && go build -o ../../bin/magpie

falcon:
	cd cmd/verification/falcon && go build -o ../../../bin/falcon

ctrlh:
	cd cmd/ctrlh && go build -o ../../bin/ctrlh

verification:
	cd cmd/verification/falcon && go build -o ../../../bin/falcon
	cd cmd/verification/radiustest && go build -o ../../../bin/radiustest
	cd cmd/verification/inputtest && go build -o ../../../bin/inputtest
	cd cmd/verification/fotaclient && go build -o ../../../bin/fotaclient
	cd cmd/verification/messagereceiver && go build -o ../../../bin/messagereceiver

# Check requires a few extra tools. Google how to install.
check: lint vet staticcheck revive

lint:
	golint ./...

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

revive:
	revive ./...

test:
	go test ./...

test_cover:
	# Run go tool cover -html=unittests.cover to see coverage for all unit tests
	go test ./... -cover -coverprofile=unittests.cover -coverpkg=github.com/eesrc/horde/pkg/...

test_race:
	go test ./... -race

test_all: test_cover test_race

benchmark:
	cd output && go test -bench .

dep_protoc:
	go get -u github.com/golang/protobuf/protoc-gen-go

run: clean build
	./bin/horde --log-type=plus \
		--log-level=debug \
		--db-create-schema=true \
		--connect-enabled=true \
		--connect-emulate=true \
		--http-endpoint=localhost:8080 \
		--enable-local-outputs \
		--launch-data-storage \
		--monitoring-endpoint=localhost:10000 \
		--management-endpoint=127.0.0.1:1234 \
		--fota-firmware-endpoint=coap://127.0.0.1:5683/fw \
		--db-connection-string="file::memory:?cache=shared" \
		--db-type=sqlite3 \
		--fota-lwm2m-poll-interval=1s \
		${HORDE_ARGS}

run_postgres: test horde
	./bin/horde --log-type=plus --log-level=debug \
		--connect-enabled=true --connect-emulate=true \
		--http-endpoint=localhost:8080 \
		--db-type=postgres \
		--db-connection-string=postgres://localhost/horde?sslmode=disable \
		--enable-local-outputs \
		--launch-data-storage \
		--data-storage-sql-connection-string=postgres://localhost/horde?sslmode=disable \
		--data-storage-sql-type=postgres \
		--monitoring-endpoint=localhost:10000 \
		--management-endpoint=127.0.0.1:1234 \
		--audit-log=true \
    	--fota-firmware-endpoint=coap://127.0.0.1:5683/fw \
		--fota-lwm2m-timeout 5s \
		--fota-download-timeout 30s \
		${HORDE_ARGS}

run_sqlite: test build
	./bin/horde --log-type=plus --log-level=debug \
		--connect-enabled=true \
		--connect-emulate=true \
		--enable-local-outputs \
		--db-type=sqlite3 \
		--db-connection-string="data/horde.db?_foreign_keys=1" \
		--db-create-schema=true \
		--http-endpoint=localhost:8080 \
		--launch-data-storage \
		--data-storage-sql-connection-string="data/magpie.db?_foreign_keys=1" \
		--data-storage-sql-type=sqlite3 \
		--management-endpoint=127.0.0.1:1234 \
		--fota-firmware-endpoint=coap://127.0.0.1:5683/fw \
		--fota-lwm2m-timeout 5s \
		--fota-download-timeout 30s \
		 ${HORDE_ARGS}

sqlite_schema:
	./bin/horde --log-type=plain \
		--db-type=sqlite3 \
		--db-connection-string="data/horde.db" \
		--db-create-schema --log-level=debug -log-type=plain --db-exit-after-create

run_frontend: clean build
	./bin/horde --log-level=debug \
		--log-type=plain \
		--enable-local-outputs=true \
		--connect-enabled=true \
		--connect-client-id=telenordigital-connectexample-web \
		--connect-password="" \
		--connect-host=connect.staging.telenordigital.com \
		--db-type=postgres \
		--db-connection-string="postgres://localhost/horde?sslmode=disable" \
		--connect-login-target=http://localhost:9000/ \
		--connect-logout-target=http://localhost:9000/ \
		--launch-data-storage \
		 ${HORDE_ARGS}

run_auth: clean build
	./bin/horde --log-level=debug \
		--connect-enabled=true \
		--connect-client-id=telenordigital-connectexample-web \
		--connect-password="" \
		--connect-host=connect.staging.telenordigital.com \
		--db-type=postgres \
		--db-connection-string="postgres://localhost/horde?sslmode=disable" \
		--github-client-id=${GH_CLIENT} \
		--github-client-secret=${GH_SECRET} \
		--github-db-connection-string="postgres://localhost/horde?sslmode=disable" \
		--github-db-driver=postgres \
		--launch-data-storage \
		--data-storage-sql-connection-string="postgres://localhost/horde?sslmode=disable" \
		--data-storage-sql-type=postgres \
		 ${HORDE_ARGS}

postgres: clean build
	mkdir -p data/postgresdb
	pg_ctl initdb -D data/postgresdb
	pg_ctl start -D data/postgresdb -l data/postgres.log
	POSTGRES=postgres://localhost/postgres?sslmode=disable make test
	pg_ctl stop -D data/postgresdb
	rm -fR data/postgresdb

clean_postgres:
	pg_ctl stop -D data/postgresdb
	rm -fR data/postgresdb

builds: linux macos

macos:
	cd cmd/horde                        && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/horde.darwin
	cd cmd/magpie                       && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/magpie.darwin
	cd cmd/ctrlh                        && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/ctrlh.darwin
	cd cmd/ingress/horde-udp               && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/horde-udp.darwin
	cd cmd/ingress/horde-coap              && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/horde-coap.darwin
	cd cmd/ingress/horde-radius            && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/horde-radius.darwin
	cd cmd/verification/radiustest      && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/radiustest.darwin
	cd cmd/verification/inputtest       && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/inputtest.darwin
	cd cmd/verification/fotaclient      && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/fotaclient.darwin
	cd cmd/verification/messagereceiver && GOOS=darwin GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/messagereceiver.darwin

linux:
	docker run --rm -it -v ${GOPATH}:/go -v $(CURDIR):/horde -w /horde/cmd ee/cross:go1.14.2 sh -c '\
		cd horde                           && echo horde...           && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/horde.linux \
		&& cd ../magpie                  && echo magpie...          && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/magpie.linux \
		&& cd ../ctrlh                   && echo ctrlh...           && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/ctrlh.linux \
		&& cd ../ingress/horde-udp          && echo horde-udp...       && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/horde-udp.linux \
		&& cd ../horde-coap              && echo horde-coap...      && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/horde-coap.linux \
		&& cd ../horde-radius            && echo horde-radius...    && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/horde-radius.linux \
		&& cd ../../verification/radiustest && echo radiustest...   && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/radiustest.linux \
		&& cd ../inputtest               && echo inputtest...       && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/inputtest.linux \
		&& cd ../fotaclient              && echo fotaclient...      && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/fotaclient.linux \
		&& cd ../messagereceiver         && echo messagereceiver... && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../bin/messagereceiver.linux \
		'
# Verification tools. Only full releases will build these tools.
linux_verification:
	docker run --rm -it -v ${GOPATH}:/go -v $(CURDIR):/horde -w /horde/cmd ee/cross:go1.14.2 sh -c '\
		cd verification/radiustest && echo radiustest... && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/radiustest.linux \
		&& cd ../datagenerator     && echo datagenerator... && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/datagenerator.linux \
		&& cd ../cachetest         && echo cachetest... && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/cachetest.linux \
		&& cd ../inputtest         && echo inputtest... && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/inputtest.linux \
		&& cd ../messagereceiver   && echo messagereceiver... && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/messagereceiver.linux'

linux_native:
	cd cmd/horde                        && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/horde.linux
	cd cmd/magpie                       && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/magpie.linux
	cd cmd/ctrlh                        && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../bin/ctrlh.linux
	cd cmd/ingress/horde-udp               && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../../bin/horde-udp.linux
	cd cmd/ingress/horde-coap              && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../../bin/horde-coap.linux
	cd cmd/ingress/horde-radius            && GOOS=linux GOARCH=amd64 go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../../bin/horde-radius.linux
	cd cmd/verification/radiustest      && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/radiustest.linux
	cd cmd/verification/inputtest       && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/inputtest.linux
	cd cmd/verification/fotaclient      && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/fotaclient.linux
	cd cmd/verification/messagereceiver && GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o ../../../bin/messagereceiver.linux

rel:
	cd cmd/horde             && go build -ldflags "$(ldflags)" -o ../../bin/horde
	cd cmd/magpie            && go build -ldflags "$(ldflags)" -o ../../bin/magpie
	cd cmd/ctrlh             && go build -ldflags "$(ldflags)" -o ../../bin/ctrlh
	cd cmd/ingress/horde-udp    &&  go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../../bin/horde-udp
	cd cmd/ingress/horde-coap   &&  go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../../bin/horde-coap
	cd cmd/ingress/horde-radius &&  go build -ldflags "$(ldflags)" -installsuffix cgo -o ../../../../bin/horde-radius

generate:
	go generate ./...

