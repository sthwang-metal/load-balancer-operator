package srv

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	lbapi "go.infratographer.com/load-balancer-api/pkg/client"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// OperatorManagedLabelKey is the label key for the operator managed namespaces
var OperatorManagedLabelKey = "com.infratographer.lb-operator/managed"

func (s *Server) ReconcileTimer(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return errMissingReconcilerInterval
	}

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			s.Logger.Info("reconciler done")

			return nil
		case <-ticker.C:
			s.Logger.Info("starting reconciler run")

			if err := s.Reconcile(ctx); err != nil {
				return err
			}
		}
	}
}

// Reconcile will reconcile out of sync load balancers
func (s *Server) Reconcile(ctx context.Context) error {
	s.Logger.Info("starting reconciler")

	if len(s.Locations) < 1 {
		s.Logger.Warnf("missing location, %+v", s.Locations)
		return errMissingLocation
	}

	var lbs []lbapi.LoadBalancer

	for idx := range s.Locations {
		lbsInLocation, err := s.APIClient.GetLoadBalancersByLocation(ctx, s.Locations[idx])
		if err != nil {
			return err
		}

		lbs = append(lbs, lbsInLocation...)
	}

	var expectedLBs []string
	for _, lb := range lbs {
		expectedLBs = append(expectedLBs, lb.ID)
	}

	kc, err := kubernetes.NewForConfig(s.KubeClient)
	if err != nil {
		return err
	}

	// ensure only namespaces the operator is managing are returned
	namespaceList, err := kc.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: OperatorManagedLabelKey,
	})
	if err != nil {
		return err
	}

	s.Logger.Debugf("expected lbs: %+v\n", expectedLBs)

	var actualLBs []string

	for _, ns := range namespaceList.Items {
		lbID, err := unhashNamespace(ns.Name)
		if err != nil {
			return err
		}

		actualLBs = append(actualLBs, lbID)
	}

	s.Logger.Debugf("actual lbs: %+v\n", actualLBs)

	// add load balancers
	toAddLBs := difference(expectedLBs, actualLBs)
	s.Logger.Infof("LBs to add: %+v", toAddLBs)

	if err = s.addLBs(ctx, toAddLBs); err != nil {
		s.Logger.Error("Error reconciling (adding) missing lbs:", err)
	}

	// remove load balancers
	toRemoveLBs := difference(actualLBs, expectedLBs)
	s.Logger.Infof("LBs to remove: %+v", toRemoveLBs)

	if err = s.removeLBs(ctx, toRemoveLBs); err != nil {
		s.Logger.Error("Error reconciling (removing) extra lbs:", err)
	}

	s.Logger.Info("finished reconciler")

	return nil
}

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))

	for _, x := range b {
		mb[x] = struct{}{}
	}

	var diff []string

	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func unhashNamespace(ns string) (string, error) {
	decodedBytes, err := hex.DecodeString(ns)

	if err != nil {
		return "", err
	}

	lbID := string(decodedBytes)

	_, err = gidx.Parse(lbID)
	if err != nil {
		return "", err
	}

	return lbID, nil
}

func (s *Server) removeLBs(ctx context.Context, lbs []string) error {
	var e error

	var evt = events.DeleteChangeType

	for _, lbID := range lbs {
		lb, err := s.getLoadBalancer(ctx, lbID, evt)
		if err != nil {
			s.Logger.Debug("Error getting lb from api: ", err)
			e = errors.Join(e, err)

			continue
		}

		task := &lbTask{
			lb:  lb,
			ctx: ctx,
			evt: string(evt),
			srv: s,
		}

		s.Runner.writer <- task
	}

	return nil
}

func (s *Server) addLBs(ctx context.Context, lbs []string) error {
	var e error

	var evt = events.CreateChangeType

	for _, lbID := range lbs {
		lb, err := s.getLoadBalancer(ctx, lbID, evt)

		if err != nil {
			s.Logger.Warn("Error getting lb from api: ", err)
			e = errors.Join(e, err)

			continue
		}

		task := &lbTask{
			lb:  lb,
			ctx: ctx,
			evt: string(evt),
			srv: s,
		}

		s.Runner.writer <- task
	}

	return e
}

func (s *Server) getLoadBalancer(ctx context.Context, lbID string, evt events.ChangeType) (*loadBalancer, error) {
	l := new(loadBalancer)

	parsedID, err := gidx.Parse(lbID)
	if err != nil {
		return nil, err
	}

	l.loadBalancerID = parsedID
	l.lbType = typeLB

	// if lb is deleted, only need the lbID
	if evt == events.DeleteChangeType {
		return l, nil
	}

	ctx, span := otel.Tracer(instrumentationName).Start(ctx, "getLoadBalancer")
	defer span.End()

	if l.lbType != typeNoLB {
		data, err := s.APIClient.GetLoadBalancer(ctx, lbID)
		if err != nil {
			s.Logger.Debugw("unable to get loadbalancer from API", "error", err, "loadBalancer", lbID)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return nil, err
		}

		l.lbData = data
	}

	return l, nil
}
