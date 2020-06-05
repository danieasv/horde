# Release test scripts

This directory contains shell scripts to do automatic tests on releases.

There *will* be a mess if the tests fail; the easiest is probably to delete
the entire test stack and start over.

Run from the root of the repository with `VERSION=n.n.n bash scripts/release-test/release-test.sh`.
This will build the docker images, launch a stack with all services and then run the various
tests.

If everything goes pear-shaped run `VERSION=n.n.n docker-compose down -v` to remove the
volumes, then `VERSION=n.n.n docker-compose up` to rebuild the stack.


## A word of warning

Docker-compose is a bit clever and caches the docker containers between invocations
if you are building a new version with the same name (f.e. "develop"), shut down the
docker compose stack with `docker-compose down -v` to get it to pick up the
latest images, otherwise it will keep on using the old versions.

## Clean up database

If the test fails you can purge the database with the `dbcleanup.sql` script. Run it with
`psql -h localhost -p 5432 -U postgres -f scripts/release-test/dbcleanup.sql`

The default password is "dbpass" in case you don't want to dig around in docker configuration files.