/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	csfov1alpha1 "github.com/sijoma/cloud-storage-file-operator/api/v1alpha1"
	"github.com/sijoma/cloud-storage-file-operator/pkg/gcp"
	"github.com/sijoma/cloud-storage-file-operator/pkg/resources"
)

// FolderReconciler reconciles a Folder object
type FolderReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	gcpClient *gcp.Client
}

//+kubebuilder:rbac:groups=csfo.sijoma.dev,resources=folders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=csfo.sijoma.dev,resources=folders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=csfo.sijoma.dev,resources=folders/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *FolderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues(
		"folder", req.NamespacedName.Name,
		"namespace", req.Namespace,
	)
	logger.Info("reconciling started")

	// populate this CRD
	folderCR := new(csfov1alpha1.Folder)
	if err := r.Get(ctx, req.NamespacedName, folderCR); err != nil {
		// do not requeue "not found" errors
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	folder, err := r.gcpClient.CreateManagedFolder(
		ctx,
		folderCR.Name+"/"+folderCR.Namespace+"/",
		folderCR.Spec.BucketName,
	)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("folder created/found", "name", folder)

	kubernetesSAName := folderCR.Name + "-owner"
	kubernetesNamespace := folderCR.Namespace
	gcpSAName := folderCR.Name + "-" + folderCR.Namespace

	// Create Service Account with workload identity
	account, err := r.gcpClient.CreateServiceAccount(ctx, gcpSAName,
		kubernetesSAName, kubernetesNamespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("created service account", "name", account.Name)

	principal := fmt.Sprintf("serviceAccount:%s", account.Email)
	const role = "roles/storage.folderAdmin"
	err = r.gcpClient.GrantRoleOnFolder(ctx, folder, folderCR.Spec.BucketName, role, principal)
	if err != nil {
		return ctrl.Result{}, err
	}

	k8sSA, err := resources.ServiceAccountWIFEnabled(ctx, r.Client,
		folderCR, kubernetesSAName, kubernetesNamespace, gcpSAName, account.ProjectId)
	if err != nil {
		return ctrl.Result{}, err
	}

	// We now have:
	// - IAM Service account
	// - IAM workload identity user
	// - Binding of Service account to ManagedFolder with "roles/storage.folderAdmin"
	// - Kubernetes SA with annotation

	folderCR.Status.ServiceAccountName = k8sSA.Name
	folderCR.Status.Email = account.Email
	folderCR.Status.Folder = folder
	err = r.Status().Update(ctx, folderCR)
	if err != nil {
		logger.Error(err, "failed to update folder status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FolderReconciler) SetupWithManager(mgr ctrl.Manager, gcpProjectID string) error {
	ctx := context.Background()
	gcpClient, err := gcp.NewGCPClient(ctx, gcpProjectID)
	if err != nil {
		return fmt.Errorf("could not create GCP client: %w", err)
	}
	r.gcpClient = gcpClient

	return ctrl.NewControllerManagedBy(mgr).
		For(&csfov1alpha1.Folder{}).
		Complete(r)
}
