package srv

import (
	"context"
	"encoding/json"
	"time"

	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"

	metastatus "go.infratographer.com/load-balancer-api/pkg/metadata"
	metacli "go.infratographer.com/metadata-api/pkg/client"

	"go.infratographer.com/load-balancer-operator/internal/config"
)

// LoadBalancerStatusUpdate updates the state of a load balancer in the metadata service
func (s Server) LoadBalancerStatusUpdate(ctx context.Context, loadBalancerID gidx.PrefixedID, status *metastatus.LoadBalancerStatus) error {
	// publish event even if metadata endpoint is not configured
	if err := s.publishLoadBalancerMetadata(ctx, loadBalancerID, status); err != nil {
		s.Logger.Warnf("Failed to publish event: %w", err)
	}

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

func (s Server) publishLoadBalancerMetadata(ctx context.Context, loadBalancerID gidx.PrefixedID, status *metastatus.LoadBalancerStatus) error {
	eventType := "metadata"

	subject := "load-balancer"

	switch status.State {
	case metastatus.LoadBalancerStateDeleted:
		subject += ".deleted"
	case metastatus.LoadBalancerStateActive:
		subject += ".active"
	default:
		s.Logger.Debugf("skipping publish message for status: %s", string(status.State))
		return nil
	}

	msg := events.EventMessage{
		EventType: eventType,
		SubjectID: loadBalancerID,
		Source:    config.AppConfig.Metadata.Source,
		Timestamp: time.Now().UTC(),
	}

	// full topic = cfg.PublisherPrefix + "events" + eventType + subject
	if _, err := s.EventsConnection.PublishEvent(ctx, subject, msg); err != nil {
		return err
	}

	return nil
}
