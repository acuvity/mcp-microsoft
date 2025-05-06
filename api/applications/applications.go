package applications

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/acuvity/mcp-microsoft/baggage"
	"github.com/acuvity/mcp-microsoft/collection"
	"github.com/mark3labs/mcp-go/mcp"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/microsoftgraph/msgraph-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

func init() {
	// Application Tool is a tool that interacts with microsoft for application APIs.
	collection.RegisterTool(
		collection.Tool{
			Name: "applications",
			Tool: mcp.NewTool("applications",
				mcp.WithDescription("Interact with Microsoft Graph API for application operations"),
				mcp.WithString("name",
					mcp.Description("The name of the application. If not provided, all applications will be returned."),
				),
			),
			Processor: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

				client := baggage.BaggageFromContext(ctx).(*msgraphsdk.GraphServiceClient)
				if client == nil {
					return mcp.NewToolResultError("client not found"), nil
				}

				params := &applications.ApplicationsRequestBuilderGetQueryParameters{}
				if name, ok := request.Params.Arguments["name"]; ok {
					params.Filter = to.Ptr("displayName eq '" + name.(string) + "'")
				}
				// Get the list of applications
				jsonData, err := Get(ctx, client, params)
				if err != nil {
					return mcp.NewToolResultError("failed to get applications"), err
				}

				return mcp.NewToolResultText(string(jsonData)), nil
			},
		},
	)
}

// Get retrieves all applications from Microsoft Graph and returns their preferred names or IDs.
func Get(ctx context.Context, client *msgraphsdk.GraphServiceClient, params *applications.ApplicationsRequestBuilderGetQueryParameters) ([]byte, error) {

	if params == nil {
		params = &applications.ApplicationsRequestBuilderGetQueryParameters{}
	}

	requestConfig := &applications.ApplicationsRequestBuilderGetRequestConfiguration{
		QueryParameters: params,
	}

	result, err := client.Applications().Get(ctx, requestConfig)
	if err != nil {
		return nil, err
	}

	// Get the applications from the result
	applications := result.GetValue()
	if applications == nil {
		return nil, err
	}

	// Create a map to store the JSON-friendly data
	applicationsData := make(map[string]interface{})

	// Convert each application to a map of attributes
	for _, application := range applications {
		id, applicationData := convertApplicationToMap(application)
		applicationsData[id] = applicationData
	}

	// Use PageIterator to iterate through all applications
	pageIterator, err := msgraphcore.NewPageIterator[models.Applicationable](result, client.GetAdapter(), models.CreateApplicationCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	err = pageIterator.Iterate(context.Background(), func(application models.Applicationable) bool {
		id, applicationData := convertApplicationToMap(application)
		applicationsData[id] = applicationData
		return true
	})
	if err != nil {
		return nil, err
	}

	// Convert the user data to JSON
	return json.MarshalIndent(applicationsData, "", "  ")
}

// convertApplicationToMap converts a application model to a map with all attributes
func convertApplicationToMap(application models.Applicationable) (string, map[string]interface{}) {
	appId := ""
	appMap := make(map[string]interface{})

	if application.GetId() != nil {
		appId = *application.GetId()
		appMap["id"] = appId
	}
	if application.GetDisplayName() != nil {
		appMap["displayName"] = *application.GetDisplayName()
	}
	if application.GetAppId() != nil {
		appMap["appId"] = *application.GetAppId()
	}
	if application.GetPublisherDomain() != nil {
		appMap["publisherDomain"] = *application.GetPublisherDomain()
	}
	if application.GetCreatedDateTime() != nil {
		appMap["createdDateTime"] = application.GetCreatedDateTime().Format(time.RFC3339)
	}
	if application.GetApplicationTemplateId() != nil {
		appMap["applicationTemplateId"] = *application.GetApplicationTemplateId()
	}
	if application.GetDefaultRedirectUri() != nil {
		appMap["defaultRedirectUri"] = *application.GetDefaultRedirectUri()
	}
	if application.GetDescription() != nil {
		appMap["description"] = *application.GetDescription()
	}
	if application.GetDisabledByMicrosoftStatus() != nil {
		appMap["disabledByMicrosoftStatus"] = *application.GetDisabledByMicrosoftStatus()
	}
	if application.GetGroupMembershipClaims() != nil {
		appMap["groupMembershipClaims"] = *application.GetGroupMembershipClaims()
	}
	if application.GetIsDeviceOnlyAuthSupported() != nil {
		appMap["isDeviceOnlyAuthSupported"] = *application.GetIsDeviceOnlyAuthSupported()
	}
	if application.GetIsFallbackPublicClient() != nil {
		appMap["isFallbackPublicClient"] = *application.GetIsFallbackPublicClient()
	}
	if application.GetNotes() != nil {
		appMap["notes"] = *application.GetNotes()
	}
	if application.GetOauth2RequirePostResponse() != nil {
		appMap["oauth2RequirePostResponse"] = *application.GetOauth2RequirePostResponse()
	}
	if application.GetSamlMetadataUrl() != nil {
		appMap["samlMetadataUrl"] = *application.GetSamlMetadataUrl()
	}
	if application.GetServiceManagementReference() != nil {
		appMap["serviceManagementReference"] = *application.GetServiceManagementReference()
	}
	if application.GetSignInAudience() != nil {
		appMap["signInAudience"] = *application.GetSignInAudience()
	}
	if application.GetTags() != nil {
		appMap["tags"] = application.GetTags()
	}
	if application.GetTokenEncryptionKeyId() != nil {
		appMap["tokenEncryptionKeyId"] = application.GetTokenEncryptionKeyId().String()
	}
	if application.GetUniqueName() != nil {
		appMap["uniqueName"] = *application.GetUniqueName()
	}

	// Encode logo if available
	if logo := application.GetLogo(); len(logo) > 0 {
		appMap["logo"] = base64.StdEncoding.EncodeToString(logo)
	}

	// Include summaries of complex types if needed
	if appApi := application.GetApi(); appApi != nil {
		appMap["api"] = "ApiApplication present"
	}
	if web := application.GetWeb(); web != nil {
		appMap["web"] = "WebApplication present"
	}
	if spa := application.GetSpa(); spa != nil {
		appMap["spa"] = "SpaApplication present"
	}
	if cert := application.GetCertification(); cert != nil {
		appMap["certification"] = "Certification present"
	}
	if info := application.GetInfo(); info != nil {
		appMap["info"] = "InformationalUrl present"
	}
	if verifiedPublisher := application.GetVerifiedPublisher(); verifiedPublisher != nil {
		appMap["verifiedPublisher"] = "VerifiedPublisher present"
	}

	// AdditionalData can include custom properties added at runtime
	if additional := application.GetAdditionalData(); additional != nil {
		for k, v := range additional {
			appMap[k] = v
		}
	}

	return appId, appMap
}
