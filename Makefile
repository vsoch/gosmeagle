all:
	gofmt -s -w .
	go build -o gosmeagle
	
build:
	go build -o gosmeagle
	
run:
	go run main.go
