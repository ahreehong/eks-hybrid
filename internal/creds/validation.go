package creds

import (
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/eks-hybrid/internal/api"
	"github.com/aws/eks-hybrid/internal/iamrolesanywhere"
	"github.com/aws/eks-hybrid/internal/retrier"
	"github.com/aws/eks-hybrid/internal/ssm"
	"github.com/aws/eks-hybrid/internal/validation"
)

// Validations returns a list of validations for the given node configuration
func Validations(config aws.Config, node *api.NodeConfig) []validation.Validation[*api.NodeConfig] {
	return ValidationsWithPoller(config, node, retrier.NewDefaultPoller())
}

// ValidationsWithPoller returns a list of validations for the given node configuration using the provided poller
func ValidationsWithPoller(config aws.Config, node *api.NodeConfig, poller retrier.Poller) []validation.Validation[*api.NodeConfig] {
	if node.IsSSM() {
		return []validation.Validation[*api.NodeConfig]{
			validation.New("ssm-api-network", ssm.NewAccessValidatorWithPoller(config, poller).Run),
		}
	}
	if node.IsIAMRolesAnywhere() {
		return []validation.Validation[*api.NodeConfig]{
			validation.New("iam-ra-api-network", iamrolesanywhere.NewAccessValidatorWithPoller(config, poller).Run),
		}
	}

	return nil
}
