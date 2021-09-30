all:
	gofmt -s -w .
	go build -gcflags '-N -l' -o gosmeagle
	
build:
	go build -gcflags '-N -l' -o gosmeagle
	
run:
	go run main.go
