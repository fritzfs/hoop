package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	agentconfig "github.com/runopsio/hoop/agent/config"
	"github.com/runopsio/hoop/common/clientconfig"
	"github.com/runopsio/hoop/common/grpc"
	"github.com/runopsio/hoop/common/log"
	pb "github.com/runopsio/hoop/common/proto"
	"github.com/runopsio/hoop/common/version"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Run() {
	config, err := agentconfig.Load()
	if err != nil {
		log.Fatal(err)
	}

	clientOptions := []*grpc.ClientOptions{grpc.WithOption("origin", pb.ConnectionOriginAgent)}
	clientConfig, err := config.GrpcClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientConfig.UserAgent = fmt.Sprintf("hoopagent/%v", version.Get().Version)
	vs := version.Get()
	log.Infof("version=%v, platform=%v, mode=%v, grpc_server=%v, tls=%v - starting agent",
		vs.Version, vs.Platform, config.Mode, config.GrpcURL, !config.IsInsecure())
	switch config.Mode {
	case clientconfig.ModeAgentWebRegister:
		for i := 0; ; i++ {
			log.Infof("webregister - connecting, attempt=%v", i+1)
			client, err := grpc.Connect(clientConfig, clientOptions...)
			if err != nil {
				log.Fatalf("failed to connect to %s, err=%v", config.GrpcURL, err.Error())
			}
			err = New(client, config).Run()
			if config.IsSaved() && err != nil {
				log.Warnf("disconnected from %v, err=%v", config.GrpcURL, err)
				break
			}
			if e, ok := status.FromError(err); ok && e.Code() == codes.Unauthenticated {
				if i == 0 {
					fmt.Print("\n--------------------------------------------------------------------------\n")
					fmt.Println("VISIT THE URL BELOW TO REGISTER THE AGENT")
					fmt.Print(config.WebRegisterURL)
					fmt.Print("\n--------------------------------------------------------------------------\n")
					fmt.Println()
				}
				if i >= 30 { // ~3 minutes
					log.Warnf("timeout on registering the agent")
					break
				}
			}
			time.Sleep(time.Second * 7)
		}
	default:
		client, err := grpc.Connect(clientConfig, clientOptions...)
		if err != nil {
			log.Fatalf("failed to connect to %s, err=%v", config.GrpcURL, err.Error())
		}
		err = New(client, config).Run()
		if err != io.EOF {
			log.Fatalf("disconnected from %v, err=%v", config.GrpcURL, err.Error())
		}
		log.Warnf("disconnected from %v", config.GrpcURL)
	}
}

func parseDSN(dsn string) (apiURL string, err error) {
	if dsn == "" {
		return "", fmt.Errorf("dsn is empty")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("dsn with wrong format, err=%v", err)
	}
	if u.Scheme != "https" {
		log.Warnf("THE AGENT IS CONNECTING USING AN INSECURE SCHEME (HTTP). CONTACT THE ADMINISTRATOR")
	}
	return fmt.Sprintf("%s://%s/api/connectionapps", u.Scheme, u.Host), nil
}

func getConnectionName() (host string, err error) {
	connectionName := os.Getenv("HOOP_CONNECTION")
	if connectionName == "" {
		return "", fmt.Errorf("missing HOOP_CONNECTION env")
	}
	if strings.HasPrefix(connectionName, "env.") {
		envName := connectionName[4:]
		connectionName = os.Getenv(envName)
		if connectionName == "" {
			return "", fmt.Errorf("environment variable %q doesn't exist", envName)
		}
	}
	return connectionName, nil
}

func fetchGrpcURL(apiURL, connectionName, dsnKey string) string {
	log.Infof("waiting for connection request")
	reqBody, _ := json.Marshal(map[string]string{"connection": connectionName})
	for {
		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelFn()
		resp, err := func() (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(reqBody))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", dsnKey))
			return http.DefaultClient.Do(req)
		}()
		if err != nil {
			log.Warnf("failed connecting to api, err=%v", err)
			time.Sleep(time.Second * 10) // backoff
			continue
		}

		defer resp.Body.Close()
		switch resp.StatusCode {
		case 200:
			var data map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				log.Warnf("failed decoding response, err=%v", err)
				time.Sleep(time.Second * 10) // backoff
				continue
			}
			return fmt.Sprintf("%v", data["grpc_url"])
		case 204: // noop
		case 401:
			log.Warnf("dsn is disabled or invalid, contact the administrator")
		default:
			data, _ := io.ReadAll(resp.Body)
			log.Warnf("api responded status=%v, body=%v", resp.StatusCode, string(data))
		}
		time.Sleep(time.Second * 10) // backoff
	}
}

// RunAsSidecar is recommended when the agent needs the context of the running applications
func RunAsSidecar() {
	dsnKey := os.Getenv("HOOP_DSN")
	apiURL, err := parseDSN(dsnKey)
	if err != nil {
		log.Error(err)
		return
	}
	connectionName, err := getConnectionName()
	if err != nil {
		log.Error(err)
		return
	}

	vs := version.Get()
	log.Infof("version=%v, platform=%v, connection=%v, api-url=%v - starting agent",
		vs.Version, vs.Platform, connectionName, apiURL)
	for {
		grpcURL := fetchGrpcURL(apiURL, connectionName, dsnKey)
		isInsecure := strings.HasPrefix(grpcURL, "http://")
		log.Infof("connecting to %v, tls=%v", grpcURL, !isInsecure)
		srvAddr, err := grpc.ParseServerAddress(grpcURL)
		if err != nil {
			log.Error("failed parsing grpc address, err=%v", err)
			continue
		}
		client, err := grpc.Connect(grpc.ClientConfig{
			ServerAddress: srvAddr,
			Token:         dsnKey,
			UserAgent:     fmt.Sprintf("hoopagent/sdk-%v", version.Get().Version),
			Insecure:      isInsecure,
		},
			grpc.WithOption("origin", pb.ConnectionOriginAgent),
			grpc.WithOption("connection-name", connectionName),
		)
		if err != nil {
			log.Error("failed connecting to grpc gateway, err=%v", err)
			continue
		}
		agentConfig := &agentconfig.Config{
			Token:   dsnKey,
			GrpcURL: grpcURL,
			Mode:    clientconfig.ModeSidecar,
		}
		err = New(client, agentConfig).Run()
		if err != io.EOF {
			log.Errorf("disconnected from %v, err=%v", grpcURL, err.Error())
			continue
		}
		log.Info("disconnected from %v", grpcURL)
	}
}
