.PHONY: lint test vendor clean

export GO111MODULE=on

default: lint test

lint:
	golangci-lint run

test:
	go test -v -cover -coverprofile=coverage.txt ./...

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