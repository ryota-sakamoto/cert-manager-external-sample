package controllers

import (
	"context"

	"github.com/go-logr/logr"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
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

	return ctrl.Result{}, nil
}

func (r *CertificateRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certmanager.CertificateRequest{}).
		Complete(r)
}
