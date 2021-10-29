FROM golang:bullseye as gobase
# docker build -t ghcr.io/vsoch/gosmeagle .
WORKDIR /src/
COPY . /src/
RUN make
ENTRYPOINT ["/src/gosmeagle"]
