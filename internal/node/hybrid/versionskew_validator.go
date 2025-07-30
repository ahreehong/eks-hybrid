package hybrid

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/eks-hybrid/internal/api"
	"github.com/aws/eks-hybrid/internal/validation"
	"github.com/blang/semver/v4"
)

// Kubelet is the kubernetes node agent.
type Kubelet interface {
	// Version returns the current kubelet version
	Version() (string, error)
}

const maxVersionSkew = 3

// ValidateKubeletVersionSkew validates the version skew for kube-apiserver and kubelet.
func (hnp *HybridNodeProvider) ValidateKubeletVersionSkew(ctx context.Context, informer validation.Informer, nodeConfig *api.NodeConfig) error {
	var err error
	if hnp.cluster == nil {
		informer.Starting(ctx, kubeletVersionSkew, "Skipping kubelet version skew validation")
		informer.Done(ctx, kubeletVersionSkew, err)
		return nil
	} else {
		informer.Starting(ctx, kubeletVersionSkew, "Validating kubelet version skew")
		defer func() {
			informer.Done(ctx, kubeletVersionSkew, err)
		}()

		kubeApiServerVersion := *(hnp.cluster.Version)
		kubeletVersion, err := hnp.kubelet.Version()
		if err != nil {
			return fmt.Errorf("failed to get kubelet version: %w", err)
		}
		apiServerSemver, err := parseK8sVersion(kubeApiServerVersion)
		if err != nil {
			return fmt.Errorf("failed to parse kube-apiserver version %s: %w", kubeApiServerVersion, err)
		}

		kubeletSemver, err := parseK8sVersion(kubeletVersion)
		if err != nil {
			return fmt.Errorf("failed to parse kubelet version %s: %w", kubeletVersion, err)
		}

		if kubeletSemver.Minor > apiServerSemver.Minor {
			return fmt.Errorf("kubelet version %s is newer than kube-apiserver version %s", kubeletVersion, kubeApiServerVersion)
		}

		minorVersionDiff := int(apiServerSemver.Minor - kubeletSemver.Minor)
		if minorVersionDiff > maxVersionSkew {
			return fmt.Errorf("kubelet version %s is too old for kube-apiserver version %s; maximum supported version skew is %d minor versions",
				kubeletVersion, kubeApiServerVersion, maxVersionSkew)
		}
	}
	return nil
}

// parseK8sVersion parses Kubernetes version strings
func parseK8sVersion(version string) (semver.Version, error) {
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) == 2 {
		version += ".0"
	}

	return semver.Parse(version)
}
