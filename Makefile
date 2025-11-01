.PHONY: lint test bench vendor clean

export GO111MODULE=on

default: lint test

lint:
	golangci-lint run

test:
	go test -v -cover -coverprofile=coverage.out ./...

bench:
	go test -bench=. ./...

test_codecov:
	cat codecov.yml | curl --data-binary @- https://codecov.io/validate

yaegi_test:
	yaegi test -v .

vendor:
	go mod vendor

clean:
	rm -rf ./vendor

run_local:
	docker compose -f docker-compose.local.yml up -d --remove-orphans

restart_local:
	docker compose -f docker-compose.local.yml restart

stop_local:
	docker compose -f docker-compose.local.yml down
