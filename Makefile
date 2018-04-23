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