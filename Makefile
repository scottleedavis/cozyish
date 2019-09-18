
run:
	go run main.go

run-docker:
	docker run -p 8000:8000 scottleedavis/cozyish:latest

build-docker:
	docker build -t scottleedavis/cozyish .

deploy-docker:
	docker push scottleedavis/cozyish:latest

