
# generate protobuff

protoc --go_out=grpcgen --go_opt=paths=source_relative --go-grpc_out=grpcgen --go-grpc_opt=paths=source_relative --proto_path=../ ../proto/notifications.proto
