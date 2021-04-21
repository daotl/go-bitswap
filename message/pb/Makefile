PB = $(wildcard *.proto)
GO = $(PB:.proto=.pb.go)

all: $(GO)

%.pb.go: %.proto
		protoc --proto_path=$(GOPATH)/src:. --gogofaster_out=. $<

clean:
		rm -f *.pb.go
		rm -f *.go
build:
	protoc -I=. --proto_path=C:\Users\zjj\go\pkg\mod\github.com\gogo\protobuf@v1.3.2 --gogofaster_out=. ./message.proto