/*
Copyright 2018 The aerospike-operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admission

import (
	"context"
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
	"net/http"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike"
	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

var (
	// Enabled represents whether the validating admission webhook is enabled.
	Enabled bool
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

const (
	// serviceName is the name of the service used to expose the webhook.
	serviceName = "aerospike-operator"
	// tlsSecretName is the name of the secret that will hold tls artifacts used
	// by the webhook.
	tlsSecretName = "aerospike-operator-tls"
	// whReadyTimeout is the time to wait until the validating webhook service
	// endpoints are ready
	whReadyTimeout = time.Second * 30
)

type admissionFunc func(admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse

// ValidatingAdmissionWebhook represents a validating admission webhook.
type ValidatingAdmissionWebhook struct {
	namespace       string
	kubeClient      kubernetes.Interface
	aerospikeClient aerospikeclientset.Interface
	tlsCertificate  tls.Certificate
}

// NewValidatingAdmissionWebhook creates a ValidatingAdmissionWebhook struct that will use the specified client to
// access the API.
func NewValidatingAdmissionWebhook(
	namespace string,
	kubeClient kubernetes.Interface,
	aerospikeClient aerospikeclientset.Interface) *ValidatingAdmissionWebhook {
	return &ValidatingAdmissionWebhook{
		namespace:       namespace,
		kubeClient:      kubeClient,
		aerospikeClient: aerospikeClient,
	}
}

// Register registers the validating admission webhook.
func (s *ValidatingAdmissionWebhook) Register() error {
	// check whether a secret containing tls artifacts exists
	sec, err := s.ensureTLSSecret()
	if err != nil {
		return err
	}
	// parse the pem-encoded tls artifacts contained in the secret
	cert, err := tls.X509KeyPair(sec.Data[v1.TLSCertKey], sec.Data[v1.TLSPrivateKeyKey])
	if err != nil {
		return err
	}
	// store the tls certificate for later usage
	s.tlsCertificate = cert

	// if the admission webhook is enable, ensure it is correctly registered
	if Enabled {
		return s.ensureWebhookConfig(sec.Data[v1.TLSCertKey])
	}

	// at this point we know the admission webhook is disabled, so we should
	// warn the user and exit
	log.Warn("disabling the validating admission webhook is strongly discouraged")
	return nil
}

func (s *ValidatingAdmissionWebhook) Run(stopCh chan struct{}) {
	// create an http server and register handler functions to back the webhook
	// and the readiness probe
	mux := http.NewServeMux()
	mux.HandleFunc(aerospikeClusterWebhookPath, s.handleAerospikeCluster)
	mux.HandleFunc(aerospikeNamespaceBackupWebhookPath, s.handleAerospikeNamespaceBackup)
	mux.HandleFunc(aerospikeNamespaceRestoreWebhookPath, s.handleAerospikeNamespaceRestore)
	mux.HandleFunc(healthzPath, handleHealthz)
	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", 8443),
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{s.tlsCertificate},
		},
	}

	// shutdown the server when stopCh is closed
	go func() {
		<-stopCh
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		srv.Shutdown(ctx)
		log.Debugf("admission webhook has been shutdown")
	}()

	// start listening on the specified port
	log.Info("starting admission webhook")
	if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
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

// ensureTLSSecret generates a certificate and private key to be used for registering and serving the webhook, and
// creates a kubernetes secret containing them so they can be used by all running instances of aerospike-operator.
// in case such secret already exists, it is read and returned.
func (s *ValidatingAdmissionWebhook) ensureTLSSecret() (*v1.Secret, error) {
	// generate the certificate to use when registering and serving the webhook
	svc := fmt.Sprintf("%s.%s.svc", serviceName, s.namespace)
	now := time.Now()
	crt := x509.Certificate{
		Subject:               pkix.Name{CommonName: svc},
		NotBefore:             now,
		NotAfter:              now.Add(365 * 24 * time.Hour),
		SerialNumber:          big.NewInt(now.Unix()),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
		DNSNames:              []string{svc},
	}
	// generate the private key to use when registering and serving the webhook
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	// pem-encode the private key
	keyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  keyutil.RSAPrivateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	// self-sign the generated certificate using the private key
	sig, err := x509.CreateCertificate(rand.Reader, &crt, &crt, key.Public(), key)
	if err != nil {
		return nil, err
	}
	// pem-encode the signed certificate
	sigBytes := pem.EncodeToMemory(&pem.Block{
		Type:  cert.CertificateBlockType,
		Bytes: sig,
	})
	// create a kubernetes secret holding the certificate and private key
	sec, err := s.kubeClient.CoreV1().Secrets(s.namespace).Create(context.TODO(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tlsSecretName,
			Labels: map[string]string{
				selectors.LabelAppKey: "aerospike-operator",
			},
			Namespace: s.namespace,
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			v1.TLSCertKey:       sigBytes,
			v1.TLSPrivateKeyKey: keyBytes,
		},
	}, metav1.CreateOptions{})
	// if creation was successful, return the created secret
	if err == nil {
		return sec, nil
	}
	// a secret may already exist, in which case we should resuse it
	if errors.IsAlreadyExists(err) {
		return s.kubeClient.CoreV1().Secrets(s.namespace).Get(context.TODO(), tlsSecretName, metav1.GetOptions{})
	}
	// the secret doesn't exist, but we couldn't create it and should fail
	return nil, err
}

func (s *ValidatingAdmissionWebhook) ensureWebhookConfig(caBundle []byte) error {
	// create the webhook configuration object containing the target configuration
	vwConfig := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: aerospikeOperatorWebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.ValidatingWebhook{
			{
				Name: crd.AerospikeClusterCRDName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{
							admissionregistrationv1beta1.Create,
							admissionregistrationv1beta1.Update,
						},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups: []string{
								aerospikev1alpha2.SchemeGroupVersion.Group,
								aerospikev1alpha1.SchemeGroupVersion.Group,
							},
							APIVersions: []string{
								aerospikev1alpha2.SchemeGroupVersion.Version,
								aerospikev1alpha1.SchemeGroupVersion.Version,
							},
							Resources: []string{crd.AerospikeClusterPlural},
						},
					},
				},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Name:      serviceName,
						Namespace: s.namespace,
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
							APIGroups: []string{
								aerospikev1alpha2.SchemeGroupVersion.Group,
								aerospikev1alpha1.SchemeGroupVersion.Group,
							},
							APIVersions: []string{
								aerospikev1alpha2.SchemeGroupVersion.Version,
								aerospikev1alpha1.SchemeGroupVersion.Version,
							},
							Resources: []string{crd.AerospikeNamespaceBackupPlural},
						},
					},
				},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Name:      serviceName,
						Namespace: s.namespace,
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
							APIGroups: []string{
								aerospikev1alpha2.SchemeGroupVersion.Group,
								aerospikev1alpha1.SchemeGroupVersion.Group,
							},
							APIVersions: []string{
								aerospikev1alpha2.SchemeGroupVersion.Version,
								aerospikev1alpha1.SchemeGroupVersion.Version,
							},
							Resources: []string{crd.AerospikeNamespaceRestorePlural},
						},
					},
				},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Name:      serviceName,
						Namespace: s.namespace,
						Path:      &aerospikeNamespaceRestoreWebhookPath,
					},
					CABundle: caBundle,
				},
				FailurePolicy: &failurePolicy,
			},
		},
	}

	// attempt to register the webhook
	_, err := s.kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(context.TODO(), vwConfig, metav1.CreateOptions{})
	if err == nil {
		// registration was successful
		return nil
	}
	if !errors.IsAlreadyExists(err) {
		// the webhook doesn't exist yet but we got an unexpected error while creating
		return err
	}

	// at this point the webhook config already exists but its spec may differ.
	// as such, we must do our best to update it.

	// fetch the latest version of the config
	currCfg, err := s.kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(context.TODO(), aerospikeOperatorWebhookName, metav1.GetOptions{})
	if err != nil {
		// we've failed to fetch the latest version of the config
		return err
	}
	if reflect.DeepEqual(currCfg.Webhooks, vwConfig.Webhooks) {
		// if the specs match there's nothing to do
		return nil
	}

	// set the resulting object's spec according to the current spec
	currCfg.Webhooks = vwConfig.Webhooks

	// attempt to update the config
	if _, err := s.kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Update(context.TODO(), currCfg, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
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

// WaitReady waits for the endpoints associated with the aerospike-operator service to be ready.
func (s *ValidatingAdmissionWebhook) WaitReady() error {
	log.Info("waiting for the validating admission webhook to be ready")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ready, err := s.isReady()
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		case <-time.After(whReadyTimeout):
			return fmt.Errorf("timed out waiting for the validating admission webhook to be ready")
		}
	}
}

// isReady returns a value indicating whether the aerospike-operator service's endpoints contain at least one endpoint.
func (s *ValidatingAdmissionWebhook) isReady() (bool, error) {
	endpoints, err := s.kubeClient.CoreV1().Endpoints(s.namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if len(endpoints.Subsets) == 0 {
		return false, nil
	}
	return true, nil
}
