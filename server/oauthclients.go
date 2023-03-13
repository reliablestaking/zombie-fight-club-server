package server

import (
	"net/url"
	"os"

	"github.com/ory/hydra-client-go/client"
	hydra "github.com/ory/hydra-client-go/client"
)

type (
	HydraClient struct {
		adminClient hydra.OryHydra
	}
)

func NewHydraClientFromEnv() (*HydraClient, error) {
	adminURL, err := url.Parse(os.Getenv("HYDRA_ADMIN_URL"))
	if err != nil {
		return nil, err
	}
	hydraAdmin := hydra.NewHTTPClientWithConfig(nil, &client.TransportConfig{Schemes: []string{adminURL.Scheme}, Host: adminURL.Host, BasePath: adminURL.Path})

	client := HydraClient{
		adminClient: *hydraAdmin,
	}

	return &client, nil
}
