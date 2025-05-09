package ssm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssm_sdk "github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/aws/eks-hybrid/internal/api"
	"github.com/aws/eks-hybrid/internal/network"
	"github.com/aws/eks-hybrid/internal/retrier"
	"github.com/aws/eks-hybrid/internal/validation"
)

// CheckEndpointAccess checks if the SSM endpoint is accessible
func CheckEndpointAccess(ctx context.Context, config aws.Config) error {
	return CheckEndpointAccessWithPoller(ctx, config, retrier.NewDefaultPoller())
}

// CheckEndpointAccessWithPoller checks if the SSM endpoint is accessible using the provided poller
func CheckEndpointAccessWithPoller(ctx context.Context, config aws.Config, poller retrier.Poller) error {
	client := ssm_sdk.NewFromConfig(config)
	opts := client.Options()

	endpoint, err := opts.EndpointResolverV2.ResolveEndpoint(ctx, ssm_sdk.EndpointParameters{
		Region:   aws.String(opts.Region),
		Endpoint: opts.BaseEndpoint,
	})
	if err != nil {
		return fmt.Errorf("resolving ssm endpoint: %w", err)
	}

	err = poller.Poll(ctx, func(ctx context.Context) (bool, error) {
		err := network.CheckConnectionToHost(ctx, endpoint.URI)
		return err == nil, err
	})
	if err != nil {
		return fmt.Errorf("checking connection to ssm endpoint: %w", err)
	}

	return nil
}

// AccessValidator validates access to the AWS SSM API endpoint.
type AccessValidator struct {
	aws    aws.Config
	poller retrier.Poller
}

// NewAccessValidator returns a new AccessValidator.
func NewAccessValidator(aws aws.Config) AccessValidator {
	return AccessValidator{
		aws:    aws,
		poller: retrier.NewDefaultPoller(),
	}
}

// NewAccessValidatorWithPoller returns a new AccessValidator with a custom poller.
func NewAccessValidatorWithPoller(aws aws.Config, poller retrier.Poller) AccessValidator {
	return AccessValidator{
		aws:    aws,
		poller: poller,
	}
}

func (a AccessValidator) Run(ctx context.Context, informer validation.Informer, _ *api.NodeConfig) error {
	var err error
	informer.Starting(ctx, "ssm-endpoint-access", "Validating access to AWS SSM API endpoint")
	defer func() {
		informer.Done(ctx, "ssm-endpoint-access", err)
	}()

	if err = CheckEndpointAccessWithPoller(ctx, a.aws, a.poller); err != nil {
		err = validation.WithRemediation(err, "Ensure your network configuration allows access to the AWS SSM API endpoint")
		return err
	}

	return nil
}
