package client

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

// GetClient creates a new Microsoft Graph client using the provided credentials.
func GetClient(tenant, client, clientSecret string) (*msgraphsdk.GraphServiceClient, error) {

	// Get the credentials
	cred, err := azidentity.NewClientSecretCredential(
		tenant,       // Tenant ID
		client,       // Client ID
		clientSecret, // Client Secret
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating credentials: %v", err)
	}

	return msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
}
