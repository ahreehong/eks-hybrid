package kubernetes

import (
	"context"
	"net/url"

	"github.com/aws/eks-hybrid/internal/api"
	"github.com/aws/eks-hybrid/internal/network"
	"github.com/aws/eks-hybrid/internal/retrier"
	"github.com/aws/eks-hybrid/internal/validation"
)

// CheckConnection validates that the node can connect to the Kubernetes API endpoint
func CheckConnection(ctx context.Context, informer validation.Informer, node *api.NodeConfig) error {
	return CheckConnectionWithPoller(ctx, informer, node, retrier.NewDefaultPoller())
}

// CheckConnectionWithPoller validates that the node can connect to the Kubernetes API endpoint using the provided poller
func CheckConnectionWithPoller(ctx context.Context, informer validation.Informer, node *api.NodeConfig, poller retrier.Poller) error {
	name := "kubernetes-endpoint-access"
	var err error
	informer.Starting(ctx, name, "Validating access to Kubernetes API endpoint")
	defer func() {
		informer.Done(ctx, name, err)
	}()

	endpoint, err := url.Parse(node.Spec.Cluster.APIServerEndpoint)
	if err != nil {
		err = validation.WithRemediation(err, "Ensure the Kubernetes API server endpoint provided is correct.")
		return err
	}

	err = poller.Poll(ctx, func(ctx context.Context) (bool, error) {
		err := network.CheckConnectionToHost(ctx, *endpoint)
		return err == nil, err
	})
	if err != nil {
		err = validation.WithRemediation(err, "Ensure your network configuration allows the node to access the Kubernetes API endpoint.")
		return err
	}

	return nil
}
