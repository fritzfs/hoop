package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	pb "github.com/runopsio/hoop/common/proto"
	pbgateway "github.com/runopsio/hoop/common/proto/gateway"
	"github.com/runopsio/hoop/common/runtime"
	"github.com/runopsio/hoop/common/version"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
)

type (
	OptionKey     string
	ClientOptions struct {
		optionKey OptionKey
		optionVal string
	}
	mutexClient struct {
		grpcClient *grpc.ClientConn
		stream     pb.Transport_ConnectClient
		mutex      sync.RWMutex
	}
)

const (
	OptionConnectionName OptionKey = "connection_name"
	defaultAddress                 = "127.0.0.1:8010"
)

func WithOption(optKey OptionKey, val string) *ClientOptions {
	return &ClientOptions{optionKey: optKey, optionVal: val}
}

func Connect(serverAddress, token string, opts ...*ClientOptions) (pb.ClientTransport, error) {
	if serverAddress == "" {
		serverAddress = defaultAddress
	}
	rpcCred := oauth.NewOauthAccess(&oauth2.Token{AccessToken: token})
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithPerRPCCredentials(rpcCred),
		grpc.WithBlock(),
	}
	var contextOptions []string
	if serverAddress == defaultAddress {
		dialOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials())}
		contextOptions = append(contextOptions, "authorization", fmt.Sprintf("Bearer %s", token))
	}
	conn, err := grpc.Dial(serverAddress, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed dialing to grpc server, err=%v", err)
	}

	osmap := runtime.OS()
	ver := version.Get()
	contextOptions = append(contextOptions,
		"version", ver.Version,
		"go_version", ver.GoVersion,
		"compiler", ver.Compiler,
		"platform", ver.Platform,
		"hostname", osmap["hostname"],
		"machine_id", osmap["machine_id"],
		"kernel_version", osmap["kernel_version"],
	)
	for _, opt := range opts {
		contextOptions = append(contextOptions, []string{
			string(opt.optionKey),
			opt.optionVal}...)
	}
	requestCtx := metadata.AppendToOutgoingContext(context.Background(), contextOptions...)
	c := pb.NewTransportClient(conn)
	stream, err := c.Connect(requestCtx)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to streaming RPC server, err=%v", err)
	}

	return &mutexClient{
		grpcClient: conn,
		stream:     stream,
		mutex:      sync.RWMutex{},
	}, nil
}

func (c *mutexClient) Send(pkt *pb.Packet) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	err := c.stream.Send(pkt)
	if err != nil && err != io.EOF {
		sentry.CaptureException(fmt.Errorf("failed sending msg, type=%v, err=%v", pkt.Type, err))
	}
	return err
}

func (c *mutexClient) Recv() (*pb.Packet, error) {
	return c.stream.Recv()
}

// Close tear down the stream and grpc connection
func (c *mutexClient) Close() (error, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	streamCloseErr := c.stream.CloseSend()
	connCloseErr := c.grpcClient.Close()
	return streamCloseErr, connCloseErr
}

func (c *mutexClient) StartKeepAlive() {
	go func() {
		for {
			proto := &pb.Packet{Type: pbgateway.KeepAlive}
			if err := c.Send(proto); err != nil {
				if err != nil {
					break
				}
			}
			time.Sleep(pb.DefaultKeepAlive)
		}
	}()
}

func (c *mutexClient) StreamContext() context.Context {
	return c.stream.Context()
}
