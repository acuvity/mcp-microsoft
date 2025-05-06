package users

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/acuvity/mcp-microsoft/baggage"
	"github.com/acuvity/mcp-microsoft/collection"
	"github.com/mark3labs/mcp-go/mcp"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

func init() {
	// Application Tool is a tool that interacts with microsoft for user APIs.
	collection.RegisterTool(
		collection.Tool{
			Name: "users",
			Tool: mcp.NewTool("users",
				mcp.WithDescription("Interact with Microsoft Graph API for user operations"),
				mcp.WithString("name",
					mcp.Description("The name of the user. If not provided, all users will be returned."),
				),
			),
			Processor: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

				client := baggage.BaggageFromContext(ctx).(*msgraphsdk.GraphServiceClient)
				if client == nil {
					return mcp.NewToolResultError("client not found"), nil
				}

				params := &users.UsersRequestBuilderGetQueryParameters{}
				if name, ok := request.Params.Arguments["name"]; ok {
					params.Filter = to.Ptr("givenName eq '" + name.(string) + "'")
				}
				// Get the list of users
				jsonData, err := Get(ctx, client, params)
				if err != nil {
					return mcp.NewToolResultError("failed to get users"), err
				}

				return mcp.NewToolResultText(string(jsonData)), nil
			},
		},
	)
}

// Get retrieves all users from Microsoft Graph and returns their preferred names or IDs.
func Get(ctx context.Context, client *msgraphsdk.GraphServiceClient, params *users.UsersRequestBuilderGetQueryParameters) ([]byte, error) {

	if params == nil {
		params = &users.UsersRequestBuilderGetQueryParameters{}
	}

	requestConfig := &users.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: params,
	}

	result, err := client.Users().Get(ctx, requestConfig)
	if err != nil {
		return nil, err
	}

	// Get the users from the result
	users := result.GetValue()
	if users == nil {
		return nil, err
	}

	// Create a map to store the JSON-friendly data
	usersData := make(map[string]interface{})

	// Convert each user to a map of attributes
	for _, user := range users {
		id, userData := convertUserToMap(user)
		usersData[id] = userData
	}

	// Use PageIterator to iterate through all users
	pageIterator, err := msgraphcore.NewPageIterator[models.Userable](result, client.GetAdapter(), models.CreateUserCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	err = pageIterator.Iterate(context.Background(), func(user models.Userable) bool {
		id, userData := convertUserToMap(user)
		usersData[id] = userData
		return true
	})
	if err != nil {
		return nil, err
	}

	// Convert the user data to JSON
	return json.MarshalIndent(usersData, "", "  ")
}

// convertUserToMap converts a user model to a map with all attributes
func convertUserToMap(user models.Userable) (string, map[string]interface{}) {

	userId := ""
	userData := make(map[string]interface{})

	// Add all standard user properties
	if id := user.GetId(); id != nil {
		userId = *id
		userData["id"] = userId
	}
	if displayName := user.GetDisplayName(); displayName != nil {
		userData["displayName"] = *displayName
	}
	if userPrincipalName := user.GetUserPrincipalName(); userPrincipalName != nil {
		userData["userPrincipalName"] = *userPrincipalName
	}
	if mail := user.GetMail(); mail != nil {
		userData["mail"] = *mail
	}
	if givenName := user.GetGivenName(); givenName != nil {
		userData["givenName"] = *givenName
	}
	if surname := user.GetSurname(); surname != nil {
		userData["surname"] = *surname
	}
	if jobTitle := user.GetJobTitle(); jobTitle != nil {
		userData["jobTitle"] = *jobTitle
	}
	if mobilePhone := user.GetMobilePhone(); mobilePhone != nil {
		userData["mobilePhone"] = *mobilePhone
	}
	if officeLocation := user.GetOfficeLocation(); officeLocation != nil {
		userData["officeLocation"] = *officeLocation
	}
	if businessPhones := user.GetBusinessPhones(); businessPhones != nil {
		userData["businessPhones"] = businessPhones
	}
	if accountEnabled := user.GetAccountEnabled(); accountEnabled != nil {
		userData["accountEnabled"] = *accountEnabled
	}
	if city := user.GetCity(); city != nil {
		userData["city"] = *city
	}
	if country := user.GetCountry(); country != nil {
		userData["country"] = *country
	}
	if department := user.GetDepartment(); department != nil {
		userData["department"] = *department
	}
	if companyName := user.GetCompanyName(); companyName != nil {
		userData["companyName"] = *companyName
	}
	if streetAddress := user.GetStreetAddress(); streetAddress != nil {
		userData["streetAddress"] = *streetAddress
	}
	if postalCode := user.GetPostalCode(); postalCode != nil {
		userData["postalCode"] = *postalCode
	}
	if state := user.GetState(); state != nil {
		userData["state"] = *state
	}
	if preferredLanguage := user.GetPreferredLanguage(); preferredLanguage != nil {
		userData["preferredLanguage"] = *preferredLanguage
	}
	if employeeId := user.GetEmployeeId(); employeeId != nil {
		userData["employeeId"] = *employeeId
	}

	// Add any additional properties available through the GetAdditionalData method
	// This can include custom attributes
	if additionalData := user.GetAdditionalData(); additionalData != nil {
		for key, value := range additionalData {
			userData[key] = value
		}
	}

	return userId, userData
}
