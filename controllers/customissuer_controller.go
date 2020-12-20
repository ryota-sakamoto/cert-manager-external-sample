/*


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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/ryota-sakamoto/cert-manager-external-sample/api/v1"
)

// CustomIssuerReconciler reconciles a CustomIssuer object
type CustomIssuerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cert-manager.k8s.sakamo.dev,resources=customissuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.k8s.sakamo.dev,resources=customissuers/status,verbs=get;update;patch

func (r *CustomIssuerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("customissuer", req.NamespacedName)

	ci := &v1.CustomIssuer{}
	if err := r.Client.Get(ctx, req.NamespacedName, ci); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Error(err, "failed to get custom issuer")

		return ctrl.Result{}, err
	}

	if ci.Spec.User != "user" {
		err := fmt.Errorf("invalid user: %s", ci.Spec.User)
		log.Error(err, "failed", "namespace", ci.Namespace, "name", ci.Name)
		r.updateStatus(ctx, ci, corev1.ConditionFalse)

		return ctrl.Result{}, err
	}

	if ci.Spec.Password != "password" {
		err := fmt.Errorf("failed to login")
		log.Error(err, "failed", "namespace", ci.Namespace, "name", ci.Name)
		r.updateStatus(ctx, ci, corev1.ConditionFalse)

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.updateStatus(ctx, ci, corev1.ConditionTrue)
}

func (r *CustomIssuerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.CustomIssuer{}).
		Complete(r)
}

func (r *CustomIssuerReconciler) updateStatus(ctx context.Context, ci *v1.CustomIssuer, status corev1.ConditionStatus) error {
	if len(ci.Status.Conditions) == 0 {
		ci.Status.Conditions = append(ci.Status.Conditions, v1.CustomIssuerCondition{
			Status: status,
		})
	} else {
		ci.Status.Conditions[0].Status = status
	}

	return r.Client.Status().Update(ctx, ci)
}
