package srv

import (
	"context"
	"encoding/json"

	"go.infratographer.com/x/gidx"

	metastatus "go.infratographer.com/load-balancer-api/pkg/metadata"
	metacli "go.infratographer.com/metadata-api/pkg/client"

	"go.infratographer.com/load-balancer-operator/internal/config"
)

// LoadBalancerStatusUpdate updates the state of a load balancer in the metadata service
func (s Server) LoadBalancerStatusUpdate(ctx context.Context, loadBalancerID gidx.PrefixedID, status *metastatus.LoadBalancerStatus) error {
	if config.AppConfig.Metadata.Endpoint == "" {
		s.Logger.Warnln("metadata not configured")
		return nil
	}

	jsonBytes, err := json.Marshal(status)
	if err != nil {
		return err
	}

	if _, err := s.MetadataClient.StatusUpdate(ctx, &metacli.StatusUpdateInput{
		NodeID:      loadBalancerID.String(),
		NamespaceID: config.AppConfig.Metadata.StatusNamespaceID.String(),
		Source:      config.AppConfig.Metadata.Source,
		Data:        json.RawMessage(jsonBytes),
	}); err != nil {
		return err
	}

	return nil
}
