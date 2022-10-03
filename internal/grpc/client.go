package grpc

import ggrpc "google.golang.org/grpc"

func (_ GRPCConnector) MakeGRPCClient(url string) (*ggrpc.ClientConn, error) {
	return ggrpc.Dial(url, ggrpc.WithInsecure())
}

type GRPCConnector struct {
}
