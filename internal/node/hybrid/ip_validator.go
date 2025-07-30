package hybrid

import (
	"context"

	"github.com/aws/eks-hybrid/internal/api"
	"github.com/aws/eks-hybrid/internal/network"
	"github.com/aws/eks-hybrid/internal/validation"
)

const (
	nodeIPFlag           = "node-ip"
	hostnameOverrideFlag = "hostname-override"
)

func (hnp *HybridNodeProvider) ValidateNodeIP(ctx context.Context, informer validation.Informer, nodeConfig *api.NodeConfig) error {
	var err error
	if hnp.cluster == nil {
		informer.Starting(ctx, nodeIpValidation, "Skipping validating Node IP")
		informer.Done(ctx, nodeIpValidation, err)
		return nil
	} else {
		informer.Starting(ctx, nodeIpValidation, "Validating Node IP configuration")
		defer func() {
			informer.Done(ctx, nodeIpValidation, err)
		}()

		// Only check flags set by user in the config file to help determine IP:
		// - node-ip and hostname-override are only available as flags and cannot be set via spec.kubelet.config
		// - Hybrid nodes does not set --node-ip
		// - Hybrid nodes sets --hostname-override to either the IAM-RA Node name or the SSM instance ID, which is checked separately for DNS
		kubeletArgs := hnp.nodeConfig.Spec.Kubelet.Flags
		var iamNodeName string
		if hnp.nodeConfig.IsIAMRolesAnywhere() {
			iamNodeName = hnp.nodeConfig.Status.Hybrid.NodeName
		}
		nodeIp, err := network.GetNodeIP(kubeletArgs, iamNodeName, hnp.network)
		if err != nil {
			return err
		}

		cluster := hnp.cluster
		if err := network.ValidateClusterRemoteNetworkConfig(cluster); err != nil {
			return err
		}

		if err = network.ValidateIPInRemoteNodeNetwork(nodeIp, cluster.RemoteNetworkConfig.RemoteNodeNetworks); err != nil {
			return err
		}
	}

	return nil
}
