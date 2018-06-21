package admission

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/cert"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike"
	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikeinformers "github.com/travelaudience/aerospike-operator/pkg/client/informers/externalversions"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
)

var (
	// Enabled represents whether the validating admission webhook is enabled.
	Enabled bool
	// ServiceName is the name of the service used to expose the webhook.
	ServiceName string
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)

	aerospikeOperatorWebhookName         = fmt.Sprintf("aerospike-operator.%s", aerospike.GroupName)
	aerospikeClusterWebhookPath          = "/admission/reviews/aerospikeclusters"
	aerospikeNamespaceBackupWebhookPath  = "/admission/reviews/aerospikenamespacebackups"
	aerospikeNamespaceRestoreWebhookPath = "/admission/reviews/aerospikenamespacerestores"
	healthzPath                          = "/healthz"

	failurePolicy = admissionregistrationv1beta1.Fail
)

type admissionFunc func(admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse

// ValidatingAdmissionWebhook represents a validating admission webhook.
type ValidatingAdmissionWebhook struct {
	kubeClient              kubernetes.Interface
	aerospikeClient         aerospikeclientset.Interface
	secretsLister           corelistersv1.SecretLister
	aerospikeClustersLister aerospikelisters.AerospikeClusterLister
}

// NewValidatingAdmissionWebhook creates a ValidatingAdmissionWebhook struct that will use the specified client to
// access the API.
func NewValidatingAdmissionWebhook(
	kubeClient kubernetes.Interface,
	aerospikeClient aerospikeclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	aerospikeInformerFactory aerospikeinformers.SharedInformerFactory) *ValidatingAdmissionWebhook {
	return &ValidatingAdmissionWebhook{
		kubeClient:              kubeClient,
		aerospikeClient:         aerospikeClient,
		secretsLister:           kubeInformerFactory.Core().V1().Secrets().Lister(),
		aerospikeClustersLister: aerospikeInformerFactory.Aerospike().V1alpha1().AerospikeClusters().Lister(),
	}
}

// RegisterAndRun registers a validating admission webhook and starts the underlying server.
func (s *ValidatingAdmissionWebhook) RegisterAndRun(readyCh chan interface{}) {
	// exit early if the webhook has been disabled
	if !Enabled {
		log.Warn("the validating admission webhook has been disabled")
		close(readyCh)
		return
	}

	// use the value of the POD_NAMESPACE envvar as the namespace
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		log.Warn("POD_NAMESPACE must be set for the validating admission webhook to be registered")
		return
	}

	// generate in-memory certificate and private key to use when registering the webhook
	svc := fmt.Sprintf("%s.%s.svc", ServiceName, namespace)
	now := time.Now()
	crt := x509.Certificate{
		Subject:               pkix.Name{CommonName: svc},
		NotBefore:             now,
		NotAfter:              now.Add(365 * 24 * time.Hour),
		SerialNumber:          big.NewInt(now.Unix()),
		KeyUsage:              x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              []string{svc},
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf("failed to start admission webhook: %v", err)
		return
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &crt, &crt, key.Public(), key)
	if err != nil {
		log.Errorf("failed to start admission webhook: %v", err)
		return
	}
	caBundle := pem.EncodeToMemory(&pem.Block{
		Type:  cert.CertificateBlockType,
		Bytes: certBytes,
	})

	// create the webhook configuration object containing the target configuration
	vwConfig := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: aerospikeOperatorWebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name: crd.AerospikeClusterCRDName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{
							admissionregistrationv1beta1.Create,
							admissionregistrationv1beta1.Update,
						},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{v1alpha1.SchemeGroupVersion.Group},
							APIVersions: []string{v1alpha1.SchemeGroupVersion.Version},
							Resources:   []string{crd.AerospikeClusterPlural},
						},
					},
				},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Name:      ServiceName,
						Namespace: namespace,
						Path:      &aerospikeClusterWebhookPath,
					},
					CABundle: caBundle,
				},
				FailurePolicy: &failurePolicy,
			},
			{
				Name: crd.AerospikeNamespaceBackupCRDName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{
							admissionregistrationv1beta1.Create,
							admissionregistrationv1beta1.Update,
						},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{v1alpha1.SchemeGroupVersion.Group},
							APIVersions: []string{v1alpha1.SchemeGroupVersion.Version},
							Resources:   []string{crd.AerospikeNamespaceBackupPlural},
						},
					},
				},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Name:      ServiceName,
						Namespace: namespace,
						Path:      &aerospikeNamespaceBackupWebhookPath,
					},
					CABundle: caBundle,
				},
				FailurePolicy: &failurePolicy,
			},
			{
				Name: crd.AerospikeNamespaceRestoreCRDName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{
							admissionregistrationv1beta1.Create,
							admissionregistrationv1beta1.Update,
						},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{v1alpha1.SchemeGroupVersion.Group},
							APIVersions: []string{v1alpha1.SchemeGroupVersion.Version},
							Resources:   []string{crd.AerospikeNamespaceRestorePlural},
						},
					},
				},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Name:      ServiceName,
						Namespace: namespace,
						Path:      &aerospikeNamespaceRestoreWebhookPath,
					},
					CABundle: caBundle,
				},
				FailurePolicy: &failurePolicy,
			},
		},
	}

	// delete any existing webhook configuration with the same name
	err = s.kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(aerospikeOperatorWebhookName, &metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("failed to start admission webhook: %v", err)
		return
	}
	// create the target webhook configuration
	_, err = s.kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(vwConfig)
	if err != nil {
		log.Errorf("failed to start admission webhook: %v", err)
		return
	}

	// create an http server and register a handler function to back the webhook
	mux := http.NewServeMux()
	mux.HandleFunc(aerospikeClusterWebhookPath, s.handleAerospikeCluster)
	mux.HandleFunc(aerospikeNamespaceBackupWebhookPath, s.handleAerospikeNamespaceBackup)
	mux.HandleFunc(aerospikeNamespaceRestoreWebhookPath, s.handleAerospikeNamespaceRestore)
	mux.HandleFunc(healthzPath, handleHealthz)
	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", 8443),
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{
				{
					Certificate: [][]byte{certBytes},
					PrivateKey:  key,
				},
			},
		},
	}

	// start listening on the specified port
	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Errorf("failed to start admission webhook: %v", err)
		return
	}

	// signal that we're ready to accept connections
	close(readyCh)

	// accept incoming connections
	if err := srv.ServeTLS(l, "", ""); err != nil && err != http.ErrServerClosed {
		log.Errorf("failed to serve admission webhook: %v", err)
		return
	}
}

func (s *ValidatingAdmissionWebhook) handleAerospikeCluster(res http.ResponseWriter, req *http.Request) {
	handle(res, req, s.admitAerospikeCluster)
}

func (s *ValidatingAdmissionWebhook) handleAerospikeNamespaceBackup(res http.ResponseWriter, req *http.Request) {
	handle(res, req, s.admitAerospikeNamespaceBackup)
}

func (s *ValidatingAdmissionWebhook) handleAerospikeNamespaceRestore(res http.ResponseWriter, req *http.Request) {
	handle(res, req, s.admitAerospikeNamespaceRestore)
}

func handleHealthz(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func handle(res http.ResponseWriter, req *http.Request, admit admissionFunc) {
	var body []byte
	if req.Body != nil {
		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body = data
		}
	}

	contentType := req.Header.Get("Content-Type")
	if contentType != "application/json" {
		res.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var reviewResponse *admissionv1beta1.AdmissionResponse
	ar := admissionv1beta1.AdmissionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		reviewResponse = admissionResponseFromError(err)
	} else {
		reviewResponse = admit(ar)
	}

	response := admissionv1beta1.AdmissionReview{}
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = ar.Request.UID
	}

	ar.Request.Object = runtime.RawExtension{}
	ar.Request.OldObject = runtime.RawExtension{}

	resp, err := json.Marshal(response)
	if err != nil {
		log.Errorf("failed to write admissionresponse: %v", err)
		return
	}
	if _, err := res.Write(resp); err != nil {
		log.Errorf("failed to write admissionresponse: %v", err)
		return
	}
}

func admissionResponseFromError(err error) *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}
