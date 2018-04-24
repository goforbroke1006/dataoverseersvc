REDIS_PORT=46380

dep:
	dep ensure -v
	dep ensure -update -v

up:
	docker-compose up -d

wait5:
	sleep 5

build: up wait5 db-update dep

db-update:
	docker-compose exec db bash /code/bin/db-up.sh

run-demo:
	go run cmd/demodatasvc/main.go

redis-dump:
	redis-cli -h 127.0.0.1 -p ${REDIS_PORT} --scan --pattern '*' | sed 's/^/get /' | redis-cli -h 127.0.0.1 -p ${REDIS_PORT}