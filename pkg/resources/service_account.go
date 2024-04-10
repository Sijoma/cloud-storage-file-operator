package resources

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceAccountWIFEnabled creates a service account to be used with workload identity federation
func ServiceAccountWIFEnabled(ctx context.Context, client client.Client,
	owner client.Object, kubernetesSAName, namespace, gcpSAName, gcpProjectID string,
) (*v1.ServiceAccount, error) {
	annotationK8sSAKey := "iam.gke.io/gcp-service-account"
	annotationK8sSAValue := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", gcpSAName, gcpProjectID)

	sa := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubernetesSAName,
			Namespace: namespace,
		},
	}
	_, err := ctrl.CreateOrUpdate(ctx, client, &sa, func() error {
		if sa.Annotations == nil {
			sa.Annotations = map[string]string{}
		}
		sa.Annotations[annotationK8sSAKey] = annotationK8sSAValue

		err := ctrl.SetControllerReference(owner, &sa, client.Scheme())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &sa, nil
}
