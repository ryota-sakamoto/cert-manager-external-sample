package controllers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/go-logr/logr"
	cmutil "github.com/jetstack/cert-manager/pkg/api/util"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	metav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/jetstack/cert-manager/pkg/util/pki"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	customissuer "github.com/ryota-sakamoto/cert-manager-external-sample/api/v1"
)

type CertificateRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *CertificateRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("certificate_request", req.NamespacedName)

	cr := &certmanager.CertificateRequest{}
	if err := r.Client.Get(ctx, req.NamespacedName, cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Error(err, "failed to get certificate request")

		return ctrl.Result{}, err
	}

	log.Info("get certificate request", "cr", cr)

	if cr.Spec.IssuerRef.Group != "" && cr.Spec.IssuerRef.Group != customissuer.GroupVersion.Group {
		log.Info("not match issuerRef",
			"group", cr.Spec.IssuerRef.Group,
		)

		return ctrl.Result{}, nil
	}

	if len(cr.Status.Certificate) > 0 {
		log.Info("already complete")

		return ctrl.Result{}, nil
	}

	ci := &customissuer.CustomIssuer{}
	ciNamespaceName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      cr.Spec.IssuerRef.Name,
	}
	if err := r.Client.Get(ctx, ciNamespaceName, ci); err != nil {
		log.Error(err, "failed to get custom issuer")

		return ctrl.Result{}, err
	}

	if len(ci.Status.Conditions) == 0 || ci.Status.Conditions[0].Status != corev1.ConditionTrue {
		err := fmt.Errorf("not login yet")
		log.Error(err, "failed to get custom issuer")

		return ctrl.Result{}, err
	}

	if err := r.sign(ctx, cr); err != nil {
		log.Error(err, "failed to sign")

		return ctrl.Result{}, err
	}

	cmutil.SetCertificateRequestCondition(cr, certmanager.CertificateRequestConditionReady, metav1.ConditionTrue, certmanager.CertificateRequestReasonIssued, "Certificate issued")

	return ctrl.Result{}, r.Client.Status().Update(ctx, cr)
}

func (r *CertificateRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certmanager.CertificateRequest{}).
		Complete(r)
}

func (r *CertificateRequestReconciler) sign(ctx context.Context, cr *certmanager.CertificateRequest) error {
	csr, err := pki.DecodeX509CertificateRequestBytes(cr.Spec.Request)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{}
	secretNamespaceName := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Annotations["cert-manager.io/private-key-secret-name"],
	}
	if err := r.Client.Get(ctx, secretNamespaceName, secret); err != nil {
		return err
	}

	block, _ := pem.Decode(secret.Data["tls.key"])
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject: pkix.Name{
			Organization: []string{"k8s.sakamo.dev"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(cr.Spec.Duration.Duration),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, v := range csr.DNSNames {
		template.DNSNames = append(template.DNSNames, v)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.(*rsa.PrivateKey).PublicKey, key)
	if err != nil {
		return err
	}

	buff := bytes.Buffer{}
	if err := pem.Encode(&buff, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	cr.Status.Certificate = buff.Bytes()

	return nil
}
