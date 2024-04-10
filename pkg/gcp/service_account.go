package gcp

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (p Client) getOrCreateServiceAccount(ctx context.Context, saName, displayName string) (*iam.ServiceAccount, error) {
	logger := log.FromContext(ctx)

	email := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", saName, p.projectID)
	serviceAccountLongName := fmt.Sprintf("projects/%s/serviceAccounts/%s", p.projectID, email)

	account, err := p.client.Projects.ServiceAccounts.Get(serviceAccountLongName).Context(ctx).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) {
			if e.Code != 404 {
				return nil, fmt.Errorf("getOrCreateServiceAccount: %w", err)
			}
			logger.V(5).Info("service account not found - going to create it",
				"serviceAccount", serviceAccountLongName)
		} else {
			return nil, err
		}
	}

	if account != nil {
		return account, nil
	}

	request := &iam.CreateServiceAccountRequest{
		AccountId: saName,
		ServiceAccount: &iam.ServiceAccount{
			DisplayName: displayName,
		},
	}
	createdAccount, err := p.client.Projects.ServiceAccounts.Create("projects/"+p.projectID, request).
		Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("Projects.ServiceAccounts.Create: %w\n", err)
	}
	logger.Info("service account created", "serviceAccount", createdAccount.Name)
	return createdAccount, nil
}

func (p Client) addBindingOnSA(ctx context.Context, sa *iam.ServiceAccount, member, role string) error {
	saPolicy, err := p.client.Projects.ServiceAccounts.GetIamPolicy(sa.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("addBindingOnSA: Projects.GetIamPolicy: %w", err)
	}

	request := new(iam.SetIamPolicyRequest)
	request.Policy = saPolicy

	var binding *iam.Binding
	// Find the policy binding for role. Only one binding can have the role.
	for _, saBinding := range saPolicy.Bindings {
		if saBinding.Role == role {
			binding = saBinding
			break
		}
	}

	if binding != nil {
		// If the binding exists, adds the member to the binding
		binding.Members = append(binding.Members, member)
	} else {
		binding = &iam.Binding{}
		binding.Role = role
		binding.Members = []string{member}
		request.Policy.Bindings = append(request.Policy.Bindings, binding)
	}

	updatedPolicy, err := p.client.Projects.ServiceAccounts.SetIamPolicy(sa.Name, request).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("addBindingOnSA: Projects.SetIamPolicy: %w", err)
	}

	fmt.Printf("updated policy %s", updatedPolicy.Bindings)
	return nil
}
