generate:
	export PATH="$PATH:$(go env GOPATH)/bin"
	protoc -I proto proto/trackerinfo.proto --go_out=./gen/trackerinfov1/ --go_opt=paths=source_relative --go-grpc_out=./gen/trackerinfov1/ --go-grpc_opt=paths=source_relative