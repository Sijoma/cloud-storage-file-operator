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

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	csfov1alpha1 "github.com/sijoma/cloud-storage-file-operator/api/v1alpha1"
	"github.com/sijoma/cloud-storage-file-operator/pkg/gcp/gcs"
	"github.com/sijoma/cloud-storage-file-operator/pkg/retrievers"
)

// FileTransferReconciler reconciles a FileTransfer object
type FileTransferReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	gcs    *gcs.StorageClient
}

//+kubebuilder:rbac:groups=csfo.sijoma.dev,resources=filetransfers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=csfo.sijoma.dev,resources=filetransfers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=csfo.sijoma.dev,resources=filetransfers/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *FileTransferReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues(
		"filetransfer", req.NamespacedName.Name,
		"namespace", req.Namespace,
	)
	logger.Info("reconciling started")

	// populate this CRD
	fileTransferCR := new(csfov1alpha1.FileTransfer)
	if err := r.Get(ctx, req.NamespacedName, fileTransferCR); err != nil {
		// do not requeue "not found" errors
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var extraOpts []option.ClientOption

	if fileTransferCR.Spec.BucketSecret != nil {
		namespace := fileTransferCR.Namespace
		if fileTransferCR.Spec.BucketSecret.Namespace != "" {
			namespace = fileTransferCR.Spec.BucketSecret.Namespace
		}
		credentials, err := retrievers.Credentials(r.Client, ctx, types.NamespacedName{Name: fileTransferCR.Spec.BucketSecret.Name, Namespace: namespace})
		if err != nil {
			return ctrl.Result{}, err
		}
		extraOpts = append(extraOpts, credentials)
	}

	// Gcs client takes a context... lets see whether we can put it on the reconciler later
	gcsClient, err := gcs.NewGcsClient(ctx, extraOpts...)
	if err != nil {
		logger.Error(err, "failed to create gcs client")
		return ctrl.Result{}, err
	}
	gcsQuery := storage.Query{Prefix: fileTransferCR.Spec.Query.Prefix}

	// FindObjects
	objects, err := gcsClient.FindObjects(ctx, fileTransferCR.Spec.BucketName, gcsQuery)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Copy files if there is a destination
	if fileTransferCR.Spec.CopyDestination != nil && fileTransferCR.Status.CopyStatus != "Done" {
		err := gcsClient.CopyFiles(ctx, fileTransferCR.Spec.BucketName, gcsQuery, fileTransferCR.Spec.CopyDestination.Prefix)
		if err != nil {
			logger.Error(err, "failed to copy files")
			return ctrl.Result{}, err
		}
		fileTransferCR.Status.CopyStatus = "Done"
		logger.Info("successfully copied files")
		err = r.Status().Update(ctx, fileTransferCR)
		if err != nil {
			logger.Error(err, "failed to update status")
			return ctrl.Result{}, err
		}
	}

	if fileTransferCR.Status.FoundObjects != len(objects) {
		fileTransferCR.Status.FoundObjects = len(objects)
		logger.Info("found objects", "objectsFound", len(objects))
		// This will list all files including folders
		logger.Info("objects found", "objectsFound", objects)
		err = r.Status().Update(ctx, fileTransferCR)
		if err != nil {
			logger.Error(err, "failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FileTransferReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&csfov1alpha1.FileTransfer{}).
		Complete(r)
}
