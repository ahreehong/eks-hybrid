package kubernetes_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/aws/eks-hybrid/internal/api"
	"github.com/aws/eks-hybrid/internal/kubernetes"
	"github.com/aws/eks-hybrid/internal/retrier"
	"github.com/aws/eks-hybrid/internal/test"
	"github.com/aws/eks-hybrid/internal/validation"
)

func TestCheckConnectionSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()

	server := test.NewHTTPSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint: server.URL,
			},
		},
	}
	g.Expect(kubernetes.CheckConnection(ctx, informer, config)).To(Succeed())
}

func TestCheckConnectionInvalidURL(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()

	config := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint: "\n",
			},
		},
	}

	err := kubernetes.CheckConnection(ctx, informer, config)
	g.Expect(err).NotTo(Succeed())
	g.Expect(informer.Started).To(BeTrue())
	g.Expect(informer.DoneWith).To(HaveOccurred())
	g.Expect(validation.Remediation(informer.DoneWith)).To(Equal("Ensure the Kubernetes API server endpoint provided is correct."))
}

func TestCheckConnectionFailureWithAccess(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()

	config := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint: "https://localhost:1234",
			},
		},
	}

	err := kubernetes.CheckConnection(ctx, informer, config)
	g.Expect(err).NotTo(Succeed())
	g.Expect(informer.Started).To(BeTrue())
	g.Expect(informer.DoneWith).To(MatchError(ContainSubstring("connect: connection refused")))
	g.Expect(validation.Remediation(informer.DoneWith)).To(Equal("Ensure your network configuration allows the node to access the Kubernetes API endpoint."))
}

func TestCheckConnectionWithMockPoller(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()
	mockPoller := retrier.NewMockPoller()

	// Set up the mock to return an error
	expectedErr := errors.New("mock connection error")
	mockPoller.PollFunc = func(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
		return expectedErr
	}

	config := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint: "https://example.com",
			},
		},
	}

	// Call the function with our mock poller
	err := kubernetes.CheckConnectionWithPoller(ctx, informer, config, mockPoller)

	// Verify the error is returned and the mock was called
	g.Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
	g.Expect(mockPoller.PollCalls).To(HaveLen(1))

	// Verify the remediation message
	g.Expect(validation.Remediation(err)).To(Equal("Ensure your network configuration allows the node to access the Kubernetes API endpoint."))
}

func TestCheckConnectionWithMockPollerSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()
	mockPoller := retrier.NewMockPoller()

	// Set up the mock to succeed
	mockPoller.PollFunc = func(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
		return nil
	}

	config := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint: "https://example.com",
			},
		},
	}

	// Call the function with our mock poller
	err := kubernetes.CheckConnectionWithPoller(ctx, informer, config, mockPoller)

	// Verify success and the mock was called
	g.Expect(err).To(BeNil())
	g.Expect(mockPoller.PollCalls).To(HaveLen(1))
}
