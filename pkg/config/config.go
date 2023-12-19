package config

import (
	"crypto/tls"

	"github.com/cosmos/cosmos-sdk/codec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Addr    string `env:"GRPC_ADDR" envDefault:"grpc.constantine.archway.tech:443"`
	TLS     bool   `env:"GRPC_TLS_ENABLED" envDefault:"true"`
	Timeout int    `env:"GRPC_TIMEOUT_SECONDS" envDefault:"5"`
	Prefix  string `env:"PREFIX" envDefault:"archway"`
}

func (c Config) GRPCConn() (*grpc.ClientConn, error) {
	transportCreds := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))

	if !c.TLS {
		transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	conn, err := grpc.Dial(
		c.Addr,
		transportCreds,
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)

	return conn, err
}
