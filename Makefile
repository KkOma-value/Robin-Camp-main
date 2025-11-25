.PHONY: docker-up docker-down test-e2e

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down -v

test-e2e:
	./e2e-test.sh
