package kubernetes_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/aws/eks-hybrid/internal/api"
	"github.com/aws/eks-hybrid/internal/kubernetes"
	"github.com/aws/eks-hybrid/internal/retrier"
	"github.com/aws/eks-hybrid/internal/test"
	"github.com/aws/eks-hybrid/internal/validation"
)

type apiServerResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
	Code    int64  `json:"code"`
}

func TestMakeUnauthenticatedRequestSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Forbidden",
		Reason:  "Forbidden",
		Code:    403,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusForbidden, resp)

	g.Expect(kubernetes.MakeUnauthenticatedRequest(ctx, server.URL, server.CAPEM())).To(Succeed())
}

func TestMakeUnauthenticatedRequestBadCA(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Forbidden",
		Reason:  "Forbidden",
		Code:    403,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusForbidden, resp)

	err := kubernetes.MakeUnauthenticatedRequest(ctx, server.URL, nil)
	g.Expect(err).To(MatchError(ContainSubstring("failed to parse Cluster CA certificate")))
	g.Expect(validation.Remediation(err)).To(Equal("Ensure the Cluster CA certificate provided is correct."))
}

func TestMakeUnauthenticatedRequestBadEndpoint(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Forbidden",
		Reason:  "Forbidden",
		Code:    403,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusForbidden, resp)

	err := kubernetes.MakeUnauthenticatedRequest(ctx, "\n", server.CAPEM())
	g.Expect(err).To(HaveOccurred())
	g.Expect(validation.Remediation(err)).To(Equal("Ensure the Kubernetes API server endpoint provided is correct."))
}

func TestMakeUnauthenticatedRequestEndpointDown(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Forbidden",
		Reason:  "Forbidden",
		Code:    403,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusForbidden, resp)

	err := kubernetes.MakeUnauthenticatedRequest(ctx, "https://my-cluster.example.com", server.CAPEM())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("no such host"))
	g.Expect(validation.Remediation(err)).To(Equal("Ensure the provided Kubernetes API server endpoint is correct and the CA certificate is valid for that endpoint."))
}

func TestMakeUnauthenticatedRequestNotForbidden(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Not Found",
		Reason:  "Not Found",
		Code:    404,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusNotFound, resp)

	err := kubernetes.MakeUnauthenticatedRequest(ctx, server.URL, server.CAPEM())
	g.Expect(err).To(MatchError(ContainSubstring("expected status code from unauthenticated request 403 or 401, got 404. Message: Not Found")))
	g.Expect(validation.Remediation(err)).To(Equal("Ensure the Kubernetes API server endpoint provided is correct and the CA certificate is valid for that endpoint."))
}

func TestMakeUnauthenticatedRequestUnauthorized(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Unauthorized",
		Reason:  "Unauthorized",
		Code:    401,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusUnauthorized, resp)

	g.Expect(kubernetes.MakeUnauthenticatedRequest(ctx, server.URL, server.CAPEM())).To(Succeed())
}

func TestMakeUnauthenticatedRequestWithMockPoller(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	mockPoller := retrier.NewMockPoller()

	// Set up the mock to return an error
	expectedErr := errors.New("mock poller error")
	mockPoller.PollFunc = func(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
		return expectedErr
	}

	// Generate a valid CA certificate for the test
	ca, err := generateSelfSignedCert()
	g.Expect(err).NotTo(HaveOccurred())

	// Call the function with our mock poller
	err = kubernetes.MakeUnauthenticatedRequestWithPoller(ctx, "https://example.com", ca, mockPoller)

	// Verify the error is returned and the mock was called
	g.Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
	g.Expect(mockPoller.PollCalls).To(HaveLen(1))

	// Verify the remediation message
	g.Expect(validation.Remediation(err)).To(Equal("Ensure the provided Kubernetes API server endpoint is correct and the CA certificate is valid for that endpoint."))
}

func TestCheckUnauthenticatedAccessSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Forbidden",
		Reason:  "Forbidden",
		Code:    403,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusForbidden, resp)

	node := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint:    server.URL,
				CertificateAuthority: server.CAPEM(),
			},
		},
	}

	g.Expect(kubernetes.CheckUnauthenticatedAccess(ctx, informer, node)).To(Succeed())
	g.Expect(informer.Started).To(BeTrue())
	g.Expect(informer.DoneWith).To(BeNil())
}

func TestCheckUnauthenticatedAccessError(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()

	resp := &apiServerResponse{
		Status:  "Failure",
		Message: "Forbidden",
		Reason:  "Forbidden",
		Code:    403,
	}

	server := test.NewHTTPSServerForJSON(t, http.StatusForbidden, resp)

	ca, err := generateSelfSignedCert()
	g.Expect(err).NotTo(HaveOccurred())

	node := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint:    server.URL,
				CertificateAuthority: ca,
			},
		},
	}

	err = kubernetes.CheckUnauthenticatedAccess(ctx, informer, node)
	g.Expect(err).To(HaveOccurred())
	g.Expect(informer.Started).To(BeTrue())
	g.Expect(informer.DoneWith).To(MatchError(ContainSubstring("failed to verify certificate: x509: certificate signed by unknown authority")))
	g.Expect(validation.Remediation(informer.DoneWith)).To(Equal("Ensure the provided Kubernetes API server endpoint is correct and the CA certificate is valid for that endpoint."))
}

func TestCheckUnauthenticatedAccessWithMockPoller(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	informer := test.NewFakeInformer()
	mockPoller := retrier.NewMockPoller()

	// Set up the mock to return an error
	expectedErr := errors.New("mock poller error")
	mockPoller.PollFunc = func(ctx context.Context, conditionFunc func(ctx context.Context) (bool, error)) error {
		return expectedErr
	}

	// Generate a valid CA certificate for the test
	ca, err := generateSelfSignedCert()
	g.Expect(err).NotTo(HaveOccurred())

	node := &api.NodeConfig{
		Spec: api.NodeConfigSpec{
			Cluster: api.ClusterDetails{
				APIServerEndpoint:    "https://example.com",
				CertificateAuthority: ca,
			},
		},
	}

	// Call the function with our mock poller
	err = kubernetes.CheckUnauthenticatedAccessWithPoller(ctx, informer, node, mockPoller)

	// Verify the error is returned and the mock was called
	g.Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
	g.Expect(mockPoller.PollCalls).To(HaveLen(1))

	// Verify the informer was called
	g.Expect(informer.Started).To(BeTrue())
	g.Expect(informer.DoneWith.Error()).To(ContainSubstring(expectedErr.Error()))
}

func generateSelfSignedCert() ([]byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}

	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Example Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}), nil
}
