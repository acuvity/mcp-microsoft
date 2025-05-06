package api

type GraphClient struct {
	// GraphClient *msgraphsdk.GraphServiceClient
}

// func New() {
// 	cred, err := azidentity.NewClientSecretCredential(
// 		"1ec8ce6a-2b6d-4828-b298-be4d2eeb573b",     // Tenant ID
// 		"34c35abb-9180-46e0-87ee-d2187d31c484",     // Client ID
// 		"WH38Q~gLHaLXMuwHhzAG4jFQqvhwJx7cmgK2eae-", // Client Secret
// 		nil,
// 	)
// 	if err != nil {
// 		fmt.Printf("Error creating credentials: %v\n", err)
// 	}

// 	fmt.Println("Device code credential created successfully.")
// 	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
// 	if err != nil {
// 		fmt.Printf("Error creating client: %v\n", err)
// 		return
// 	}
// }
