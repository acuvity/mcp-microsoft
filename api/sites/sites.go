package sites

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/mark3labs/mcp-go/mcp"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/sites"
	"github.com/acuvity/mcp-microsoft/baggage"
	"github.com/acuvity/mcp-microsoft/collection"
)

func init() {
	// Site Tool is a tool that interacts with microsoft for site APIs.
	collection.RegisterTool(
		collection.Tool{
			Name: "sites",
			Tool: mcp.NewTool("sites",
				mcp.WithDescription("Interact with Microsoft Graph API for site, subsites and pages operations"),
				mcp.WithString("name",
					mcp.Description("The name of the site"),
				),
			),
			Processor: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

				client := baggage.BaggageFromContext(ctx).(*msgraphsdk.GraphServiceClient)
				if client == nil {
					return mcp.NewToolResultError("client not found"), nil
				}

				params := &sites.SitesRequestBuilderGetQueryParameters{}
				if name, ok := request.Params.Arguments["name"]; ok {
					params.Filter = to.Ptr("displayName eq '" + name.(string) + "'")
				}
				// Get the list of sites
				jsonData, err := Get(ctx, client, params)
				if err != nil {
					return mcp.NewToolResultError("failed to get sites"), err
				}

				return mcp.NewToolResultText(string(jsonData)), nil
			},
		},
	)
}

// Get retrieves all sites from Microsoft Graph and returns their preferred names or IDs.
func Get(ctx context.Context, client *msgraphsdk.GraphServiceClient, params *sites.SitesRequestBuilderGetQueryParameters) ([]byte, error) {

	if params == nil {
		params = &sites.SitesRequestBuilderGetQueryParameters{
			// Select specific properties to return
			Select: []string{"id", "displayName", "webUrl", "siteCollection", "description"},
		}
	}

	requestConfig := &sites.SitesRequestBuilderGetRequestConfiguration{
		QueryParameters: params,
	}

	result, err := client.Sites().Get(ctx, requestConfig)
	if err != nil {
		return nil, err
	}

	// Get the sites from the result
	sites := result.GetValue()
	if sites == nil {
		return nil, err
	}

	// Create a map to store the JSON-friendly data
	sitesData := make(map[string]interface{})

	// Convert each site to a map of attributes
	for _, site := range sites {
		id, siteData := convertSiteToMap(site)
		sitesData[id] = siteData
	}

	// Use PageIterator to handle pagination if there are more sites
	if result.GetOdataNextLink() != nil && *result.GetOdataNextLink() != "" {
		pageIterator, err := msgraphcore.NewPageIterator[models.Siteable](
			result,
			client.GetAdapter(),
			models.CreateSiteCollectionResponseFromDiscriminatorValue,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating page iterator: %v", err)
		}

		err = pageIterator.Iterate(context.Background(), func(site models.Siteable) bool {
			id, siteData := convertSiteToMap(site)
			sitesData[id] = siteData
			return true // Continue iteration
		})
		if err != nil {
			return nil, fmt.Errorf("error iterating over sites: %v", err)
		}
	}

	for id, site := range sitesData {

		// Handle Subsites
		subsites, err := GetSubsites(ctx, client, id)
		if err != nil {
			continue
		}
		subsiteData := make(map[string]interface{})
		for _, subsite := range subsites {
			subsiteID, subsiteInfo := convertSiteToMap(subsite)
			subsiteData[subsiteID] = subsiteInfo
		}
		site.(map[string]interface{})["subsites"] = subsiteData

		// Handle Pages
		pages, err := GetPages(ctx, client, id)
		if err != nil {
			continue
		}
		pageData := make(map[string]interface{})
		for _, page := range pages {
			pageId, pageInfo := convertSitePageToMap(page)
			content, err := getPageContent(client, id, pageId, "markdown")
			if err == nil {
				pageInfo["content"] = content
			} else {
				pageInfo["content"] = "Error fetching content"
			}
			pageData[pageId] = pageInfo
		}
		site.(map[string]interface{})["pages"] = pageData

		// Restash the site data
		sitesData[id] = site
	}

	// Convert the user data to JSON
	return json.MarshalIndent(sitesData, "", "  ")
}

// You can also create a function to get a specific site's details and subsites
func GetSubsites(ctx context.Context, client *msgraphsdk.GraphServiceClient, siteId string) ([]models.Siteable, error) {

	// Get the site's subsites
	subsitesResponse, err := client.Sites().BySiteId(siteId).Sites().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching subsites: %v", err)
	}

	subsites := subsitesResponse.GetValue()

	// Handle pagination for subsites if necessary
	if subsitesResponse.GetOdataNextLink() != nil {
		pageIterator, err := msgraphcore.NewPageIterator[models.Siteable](
			subsitesResponse,
			client.GetAdapter(),
			models.CreateSiteCollectionResponseFromDiscriminatorValue,
		)

		if err != nil {
			return subsites, fmt.Errorf("error creating page iterator for subsites: %v", err)
		}

		err = pageIterator.Iterate(ctx, func(subsite models.Siteable) bool {
			subsites = append(subsites, subsite)
			return true
		})

		if err != nil {
			return subsites, fmt.Errorf("error iterating through subsites: %v", err)
		}
	}

	return subsites, nil
}

// You can also create a function to get a specific site's details and subsites
func GetPages(ctx context.Context, client *msgraphsdk.GraphServiceClient, siteId string) ([]models.SitePageable, error) {

	// Get the site's subsites
	pagesResponse, err := client.Sites().BySiteId(siteId).Pages().GraphSitePage().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching pages: %v", err)
	}

	pages := pagesResponse.GetValue()

	// Handle pagination for subsites if necessary
	if pagesResponse.GetOdataNextLink() != nil {
		pageIterator, err := msgraphcore.NewPageIterator[models.SitePageable](
			pagesResponse,
			client.GetAdapter(),
			models.CreateSitePageCollectionResponseFromDiscriminatorValue,
		)

		if err != nil {
			return pages, fmt.Errorf("error creating page iterator for pages: %v", err)
		}

		err = pageIterator.Iterate(ctx, func(page models.SitePageable) bool {
			pages = append(pages, page)
			return true
		})

		if err != nil {
			return pages, fmt.Errorf("error iterating through pages: %v", err)
		}
	}

	return pages, nil
}

// convertSiteToMap extracts relevant fields from a Siteable into a flat map.
// It avoids deeply nested or recursive fields for simplicity and safety.
func convertSiteToMap(site models.Siteable) (string, map[string]interface{}) {
	var siteID string
	siteMap := make(map[string]interface{})

	if idPtr := site.GetId(); idPtr != nil {
		siteID = *idPtr
		siteMap["id"] = siteID
	}

	if namePtr := site.GetDisplayName(); namePtr != nil {
		siteMap["displayName"] = *namePtr
	}

	if isPersonal := site.GetIsPersonalSite(); isPersonal != nil {
		siteMap["isPersonalSite"] = *isPersonal
	}

	if analytics := site.GetAnalytics(); analytics != nil {
		siteMap["analytics"] = analytics // optionally convert to map if needed
	}

	if errorInfo := site.GetError(); errorInfo != nil {
		siteMap["error"] = errorInfo // optionally convert to map if needed
	}

	if sharepointIds := site.GetSharepointIds(); sharepointIds != nil {
		siteMap["sharepointIds"] = sharepointIds // optionally convert to map if needed
	}

	if siteCollection := site.GetSiteCollection(); siteCollection != nil {
		siteMap["siteCollection"] = siteCollection // optionally convert to map if needed
	}

	// Add AdditionalData last to allow overrides
	for k, v := range site.GetAdditionalData() {
		siteMap[k] = v
	}

	return siteID, siteMap
}

func convertSitePageToMap(page models.SitePageable) (string, map[string]interface{}) {
	var siteID string
	siteMap := make(map[string]interface{})

	// From BaseItemable
	if idPtr := page.GetId(); idPtr != nil {
		siteID = *idPtr
		siteMap["id"] = siteID
	}

	// From BaseSitePageable
	if layout := page.GetPageLayout(); layout != nil {
		siteMap["pageLayout"] = *layout
	}

	if publishingState := page.GetPublishingState(); publishingState != nil {
		siteMap["publishingState"] = publishingState
	}

	if title := page.GetTitle(); title != nil {
		siteMap["title"] = *title
	}

	// AdditionalData is included last to allow overriding
	for k, v := range page.GetAdditionalData() {
		siteMap[k] = v
	}

	return siteID, siteMap
}

// Get the content of a specific page and format as Markdown or text
func getPageContent(client *msgraphsdk.GraphServiceClient, siteId string, pageId string, format string) (string, error) {

	debugging := false

	// Set up query parameters to expand canvasLayout
	expandValue := []string{"canvasLayout"}
	queryParams := &sites.ItemPagesItemGraphSitePageRequestBuilderGetQueryParameters{
		Expand: expandValue,
	}
	requestConfig := &sites.ItemPagesItemGraphSitePageRequestBuilderGetRequestConfiguration{
		QueryParameters: queryParams,
	}

	// Get the specific page using GraphSitePage with expanded canvasLayout
	page, err := client.Sites().BySiteId(siteId).Pages().ByBaseSitePageId(pageId).GraphSitePage().Get(context.Background(), requestConfig)
	if err != nil {
		return "", fmt.Errorf("error getting page content: %v", err)
	}

	// Create a string builder for content
	var contentBuilder strings.Builder

	// Add page title if available
	if page.GetTitle() != nil {
		if format == "markdown" {
			contentBuilder.WriteString(fmt.Sprintf("## %s\n\n", *page.GetTitle()))
		} else {
			contentBuilder.WriteString(fmt.Sprintf("Title: %s\n", *page.GetTitle()))
		}
	}

	// Add page description if available
	if page.GetDescription() != nil {
		if format == "markdown" {
			contentBuilder.WriteString(fmt.Sprintf("*%s*\n\n", *page.GetDescription()))
		} else {
			contentBuilder.WriteString(fmt.Sprintf("Description: %s\n", *page.GetDescription()))
		}
	}

	if debugging {
		// Add debug information - dump the data model to see what properties are available
		contentBuilder.WriteString(fmt.Sprintf("Page ID: %s\n", *page.GetId()))

		// Print types of each property if available for debugging
		contentBuilder.WriteString("\n### Page Data Model Debug Info\n\n")
		contentBuilder.WriteString("```\n")

		// Check and print available property types from the page object
		if page.GetOdataType() != nil {
			contentBuilder.WriteString(fmt.Sprintf("OdataType: %s\n", *page.GetOdataType()))
		}

		// Print all methods available on the page object for debugging
		pageType := fmt.Sprintf("%T", page)
		contentBuilder.WriteString(fmt.Sprintf("Page Type: %s\n", pageType))
		// Get additional data from the page object
		additionalData := page.GetAdditionalData()
		if len(additionalData) > 0 {
			contentBuilder.WriteString("Additional Data Keys:\n")
			for key := range additionalData {
				contentBuilder.WriteString(fmt.Sprintf("- %s\n", key))
			}
		}

		contentBuilder.WriteString("```\n\n")
	}

	// Process canvas layout
	if page.GetCanvasLayout() != nil {
		if format == "markdown" {
			if debugging {
				contentBuilder.WriteString("---\n\n")
			}
		} else {
			contentBuilder.WriteString("--- Canvas Layout Content ---\n")
		}

		// Process horizontal sections (most common layout)
		if page.GetCanvasLayout().GetHorizontalSections() != nil {
			sections := page.GetCanvasLayout().GetHorizontalSections()

			if format == "markdown" {
				// No need to output section count in markdown format
			} else {
				contentBuilder.WriteString(fmt.Sprintf("Number of horizontal sections: %d\n", len(sections)))
			}

			// Go through each section
			for secIdx, section := range sections {
				if format == "markdown" {
					if debugging {
						contentBuilder.WriteString(fmt.Sprintf("### Section %d\n\n", secIdx+1))
						if section.GetLayout() != nil {
							contentBuilder.WriteString(fmt.Sprintf("*Layout: %s*\n\n", *section.GetLayout()))
						}
					}
				} else {
					contentBuilder.WriteString(fmt.Sprintf("\nSection %d (Layout: %s):\n", secIdx+1, *section.GetLayout()))
				}

				// Go through columns in each section
				if section.GetColumns() != nil {
					columns := section.GetColumns()

					if format == "markdown" {
						// No need to output column count in markdown
					} else {
						contentBuilder.WriteString(fmt.Sprintf("  Number of columns: %d\n", len(columns)))
					}

					// Go through each column
					for colIdx, column := range columns {
						if format == "markdown" {
							if debugging {
								contentBuilder.WriteString(fmt.Sprintf("#### Column %d\n\n", colIdx+1))
							}
						} else {
							contentBuilder.WriteString(fmt.Sprintf("  Column %d:\n", colIdx+1))
						}

						// Go through web parts in each column
						if column.GetWebparts() != nil {
							webParts := column.GetWebparts()

							if format == "markdown" {
								// No need to output webpart count in markdown
							} else {
								contentBuilder.WriteString(fmt.Sprintf("    Number of web parts: %d\n", len(webParts)))
							}

							// Extract content from each web part
							for wpIdx, webPart := range webParts {
								if format == "markdown" {
									if debugging {
										contentBuilder.WriteString(fmt.Sprintf("##### Web Part %d\n\n", wpIdx+1))
									}
								} else {
									contentBuilder.WriteString(fmt.Sprintf("    Web Part %d (Type: %T):\n", wpIdx+1, webPart))
								}

								// Try to get the content from various properties where the content might be stored
								contentFound := false
								// Different web part types have different structures
								if webPart.GetOdataType() == nil {
									if debugging {
										contentBuilder.WriteString(fmt.Sprintf("##### No OdataTType %d\n\n", wpIdx+1))
									}
								}

								backingStore := webPart.GetBackingStore()
								if backingStore != nil {
									innerHtml, err := webPart.GetBackingStore().Get("innerHtml")
									if err == nil && innerHtml != nil {
										if htmlStr, ok := innerHtml.(*string); ok {
											if format == "markdown" {
												markdownContent := htmlToMarkdown(*htmlStr)
												contentBuilder.WriteString(markdownContent)
												contentBuilder.WriteString("\n\n")
												contentFound = true
											} else {
												contentBuilder.WriteString(fmt.Sprintf("      Content: %s\n", *htmlStr))
												contentFound = true
											}
										}
									}
								}

								// Extract content from web parts
								data := webPart.GetAdditionalData()
								if data != nil {

									if !contentFound {
										// First try innerHtml which is common for text web parts
										if innerHtml, ok := data["innerHtml"]; ok {
											if htmlStr, ok := innerHtml.(string); ok {
												if format == "markdown" {
													markdownContent := htmlToMarkdown(htmlStr)
													contentBuilder.WriteString(markdownContent)
													contentBuilder.WriteString("\n\n")
													contentFound = true
												} else {
													contentBuilder.WriteString(fmt.Sprintf("      Content: %s\n", htmlStr))
													contentFound = true
												}
											}
										}
									}

									// Try getting the text property which some web parts use
									if !contentFound {
										if textContent, ok := data["text"]; ok {
											if textStr, ok := textContent.(string); ok {
												if format == "markdown" {
													contentBuilder.WriteString(textStr)
													contentBuilder.WriteString("\n\n")
													contentFound = true
												} else {
													contentBuilder.WriteString(fmt.Sprintf("      Text: %s\n", textStr))
													contentFound = true
												}
											}
										}
									}

									// Try data property where some web parts store content in a JSON structure
									if !contentFound {
										if jsonData, ok := data["data"]; ok {
											// Try to extract meaningful info from the data property
											if dataMap, ok := jsonData.(map[string]interface{}); ok {
												// Look for common content fields in the data map
												for _, field := range []string{"text", "content", "value", "description", "html"} {
													if fieldVal, exists := dataMap[field]; exists {
														if strVal, ok := fieldVal.(string); ok && strVal != "" {
															if format == "markdown" {
																if field == "html" {
																	contentBuilder.WriteString(htmlToMarkdown(strVal))
																} else {
																	contentBuilder.WriteString(strVal)
																}
																contentBuilder.WriteString("\n\n")
																contentFound = true
																break
															} else {
																contentBuilder.WriteString(fmt.Sprintf("      %s: %s\n", field, strVal))
																contentFound = true
																break
															}
														}
													}
												}
											} else if dataStr, ok := jsonData.(string); ok && dataStr != "" {
												// If data is a string, output it directly
												if format == "markdown" {
													contentBuilder.WriteString(dataStr)
													contentBuilder.WriteString("\n\n")
													contentFound = true
												} else {
													contentBuilder.WriteString(fmt.Sprintf("      Data: %s\n", dataStr))
													contentFound = true
												}
											}
										}
									}

									if debugging {
										// If no content found, log the available additional data keys for debugging
										if !contentFound {
											keys := []string{}
											for k := range data {
												keys = append(keys, k)
											}
											if format == "markdown" {
												contentBuilder.WriteString("*No readable content found for this web part.*\n\n")
												contentBuilder.WriteString(fmt.Sprintf("*Available data keys: %s*\n\n", strings.Join(keys, ", ")))
											} else {
												contentBuilder.WriteString("      No readable content found for this web part.\n")
												contentBuilder.WriteString(fmt.Sprintf("      Available data keys: %s\n", strings.Join(keys, ", ")))
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Process vertical section (single column layout)
		if page.GetCanvasLayout().GetVerticalSection() != nil {
			vertSection := page.GetCanvasLayout().GetVerticalSection()

			if format == "markdown" {
				if debugging {
					contentBuilder.WriteString("### Vertical Section\n\n")
				}
			} else {
				contentBuilder.WriteString("\nVertical Section:\n")
			}

			// Go through web parts in the vertical section
			if vertSection.GetWebparts() != nil {
				webParts := vertSection.GetWebparts()

				if format == "markdown" {
					// No need to output webpart count in markdown
				} else {
					contentBuilder.WriteString(fmt.Sprintf("  Number of web parts: %d\n", len(webParts)))
				}

				// Extract content from each web part
				for wpIdx, webPart := range webParts {
					if format == "markdown" {
						if debugging {
							contentBuilder.WriteString(fmt.Sprintf("#### Web Part %d\n\n", wpIdx+1))
						}
					} else {
						contentBuilder.WriteString(fmt.Sprintf("  Web Part %d:\n", wpIdx+1))
					}

					// Extract content from web parts
					data := webPart.GetAdditionalData()
					if data != nil {
						// Try to get the content from various properties where the content might be stored
						contentFound := false

						// First try innerHtml which is common for text web parts
						if innerHtml, ok := data["innerHtml"]; ok {
							if htmlStr, ok := innerHtml.(string); ok {
								if format == "markdown" {
									markdownContent := htmlToMarkdown(htmlStr)
									contentBuilder.WriteString(markdownContent)
									contentBuilder.WriteString("\n\n")
									contentFound = true
								} else {
									contentBuilder.WriteString(fmt.Sprintf("    Content: %s\n", htmlStr))
									contentFound = true
								}
							}
						}

						// Try getting the text property which some web parts use
						if !contentFound {
							if textContent, ok := data["text"]; ok {
								if textStr, ok := textContent.(string); ok {
									if format == "markdown" {
										contentBuilder.WriteString(textStr)
										contentBuilder.WriteString("\n\n")
										contentFound = true
									} else {
										contentBuilder.WriteString(fmt.Sprintf("    Text: %s\n", textStr))
										contentFound = true
									}
								}
							}
						}

						// Try data property where some web parts store content in a JSON structure
						if !contentFound {
							if jsonData, ok := data["data"]; ok {
								// Try to extract meaningful info from the data property
								if dataMap, ok := jsonData.(map[string]interface{}); ok {
									// Look for common content fields in the data map
									for _, field := range []string{"text", "content", "value", "description", "html"} {
										if fieldVal, exists := dataMap[field]; exists {
											if strVal, ok := fieldVal.(string); ok && strVal != "" {
												if format == "markdown" {
													if field == "html" {
														contentBuilder.WriteString(htmlToMarkdown(strVal))
													} else {
														contentBuilder.WriteString(strVal)
													}
													contentBuilder.WriteString("\n\n")
													contentFound = true
													break
												} else {
													contentBuilder.WriteString(fmt.Sprintf("    %s: %s\n", field, strVal))
													contentFound = true
													break
												}
											}
										}
									}
								} else if dataStr, ok := jsonData.(string); ok && dataStr != "" {
									// If data is a string, output it directly
									if format == "markdown" {
										contentBuilder.WriteString(dataStr)
										contentBuilder.WriteString("\n\n")
										contentFound = true
									} else {
										contentBuilder.WriteString(fmt.Sprintf("    Data: %s\n", dataStr))
										contentFound = true
									}
								}
							}
						}

						if debugging {
							// If no content found, log the available additional data keys for debugging
							if !contentFound {
								keys := []string{}
								for k := range data {
									keys = append(keys, k)
								}
								if format == "markdown" {
									contentBuilder.WriteString("*No readable content found for this web part.*\n\n")
									contentBuilder.WriteString(fmt.Sprintf("*Available data keys: %s*\n\n", strings.Join(keys, ", ")))
								} else {
									contentBuilder.WriteString("    No readable content found for this web part.\n")
									contentBuilder.WriteString(fmt.Sprintf("    Available data keys: %s\n", strings.Join(keys, ", ")))
								}
							}
						}
					}
				}
			}
		}
	}

	content := contentBuilder.String()

	// If we couldn't extract specific content
	if content == "" {
		if format == "markdown" {
			return "*No detailed content available. Use the page URL to view in browser.*", nil
		}
		return "No detailed content available. Use the page URL to view in browser.", nil
	}

	return content, nil
}

// Convert HTML content to Markdown
func htmlToMarkdown(htmlContent string) string {
	// Unescape HTML entities
	unescaped := html.UnescapeString(htmlContent)

	// Create a string builder for the result
	var result strings.Builder

	// Basic HTML to Markdown conversions - these are simplified and won't handle all HTML

	// Replace headings
	h1Regex := regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	unescaped = h1Regex.ReplaceAllString(unescaped, "# $1\n\n")

	h2Regex := regexp.MustCompile(`<h2[^>]*>(.*?)</h2>`)
	unescaped = h2Regex.ReplaceAllString(unescaped, "## $1\n\n")

	h3Regex := regexp.MustCompile(`<h3[^>]*>(.*?)</h3>`)
	unescaped = h3Regex.ReplaceAllString(unescaped, "### $1\n\n")

	h4Regex := regexp.MustCompile(`<h4[^>]*>(.*?)</h4>`)
	unescaped = h4Regex.ReplaceAllString(unescaped, "#### $1\n\n")

	// Replace paragraph tags
	pRegex := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
	unescaped = pRegex.ReplaceAllString(unescaped, "$1\n\n")

	// Replace bold tags
	boldRegex := regexp.MustCompile(`<(b|strong)[^>]*>(.*?)</\\1>`)
	unescaped = boldRegex.ReplaceAllString(unescaped, "**$2**")

	// Replace italic tags
	italicRegex := regexp.MustCompile(`<(i|em)[^>]*>(.*?)</\\1>`)
	unescaped = italicRegex.ReplaceAllString(unescaped, "*$2*")

	// Replace links
	linkRegex := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	unescaped = linkRegex.ReplaceAllString(unescaped, "[$2]($1)")

	// Replace unordered lists
	unescaped = strings.Replace(unescaped, "<ul>", "\n", -1)
	unescaped = strings.Replace(unescaped, "</ul>", "\n", -1)
	liRegex := regexp.MustCompile(`<li[^>]*>(.*?)</li>`)
	unescaped = liRegex.ReplaceAllString(unescaped, "- $1\n")

	// Replace ordered lists
	unescaped = strings.Replace(unescaped, "<ol>", "\n", -1)
	unescaped = strings.Replace(unescaped, "</ol>", "\n", -1)
	olLiRegex := regexp.MustCompile(`<li[^>]*>(.*?)</li>`)
	unescaped = olLiRegex.ReplaceAllString(unescaped, "1. $1\n")

	// Replace images
	imgRegex := regexp.MustCompile(`<img[^>]*src="([^"]*)"[^>]*alt="([^"]*)"[^>]*>`)
	unescaped = imgRegex.ReplaceAllString(unescaped, "![$2]($1)")

	// Handle tables
	tableRegex := regexp.MustCompile(`<table[^>]*>(.*?)</table>`)
	tableMatches := tableRegex.FindAllStringSubmatch(unescaped, -1)
	for _, match := range tableMatches {
		fullTableHTML := match[0]
		tableContent := match[1]

		var mdTable strings.Builder

		// Extract rows
		trRegex := regexp.MustCompile(`<tr[^>]*>(.*?)</tr>`)
		rows := trRegex.FindAllStringSubmatch(tableContent, -1)

		// Process header row
		if len(rows) > 0 {
			thRegex := regexp.MustCompile(`<th[^>]*>(.*?)</th>`)
			headerCells := thRegex.FindAllStringSubmatch(rows[0][1], -1)

			if len(headerCells) > 0 {
				// This is a header row
				for _, cell := range headerCells {
					mdTable.WriteString("| ")
					mdTable.WriteString(strings.TrimSpace(cell[1]))
					mdTable.WriteString(" ")
				}
				mdTable.WriteString("|\n")

				// Add separator row
				for i := 0; i < len(headerCells); i++ {
					mdTable.WriteString("| --- ")
				}
				mdTable.WriteString("|\n")
			} else {
				// No header cells, check for data cells in the first row for table structure
				tdRegex := regexp.MustCompile(`<td[^>]*>(.*?)</td>`)
				firstRowCells := tdRegex.FindAllStringSubmatch(rows[0][1], -1)

				// Create header based on number of columns
				for i := 0; i < len(firstRowCells); i++ {
					mdTable.WriteString("| Column ")
					mdTable.WriteString(fmt.Sprintf("%d", i+1))
					mdTable.WriteString(" ")
				}
				mdTable.WriteString("|\n")

				// Add separator row
				for i := 0; i < len(firstRowCells); i++ {
					mdTable.WriteString("| --- ")
				}
				mdTable.WriteString("|\n")
			}
		}

		// Process data rows
		for _, row := range rows {
			tdRegex := regexp.MustCompile(`<td[^>]*>(.*?)</td>`)
			cells := tdRegex.FindAllStringSubmatch(row[1], -1)

			for _, cell := range cells {
				mdTable.WriteString("| ")
				mdTable.WriteString(strings.TrimSpace(cell[1]))
				mdTable.WriteString(" ")
			}
			mdTable.WriteString("|\n")
		}

		// Replace the HTML table with the Markdown table
		unescaped = strings.Replace(unescaped, fullTableHTML, mdTable.String(), 1)
	}

	// Handle code blocks
	preRegex := regexp.MustCompile(`<pre[^>]*>(.*?)</pre>`)
	unescaped = preRegex.ReplaceAllString(unescaped, "```\n$1\n```\n\n")

	codeRegex := regexp.MustCompile(`<code[^>]*>(.*?)</code>`)
	unescaped = codeRegex.ReplaceAllString(unescaped, "`$1`")

	// Replace blockquotes
	blockquoteRegex := regexp.MustCompile(`<blockquote[^>]*>(.*?)</blockquote>`)
	unescaped = blockquoteRegex.ReplaceAllString(unescaped, "> $1\n\n")

	// Replace horizontal rules
	hrRegex := regexp.MustCompile(`<hr[^>]*>`)
	unescaped = hrRegex.ReplaceAllString(unescaped, "---\n\n")

	// Replace divs and spans with their content
	divRegex := regexp.MustCompile(`<(div|span)[^>]*>(.*?)</\\1>`)
	for divRegex.MatchString(unescaped) {
		unescaped = divRegex.ReplaceAllString(unescaped, "$2")
	}

	// Replace breaks with newlines
	brRegex := regexp.MustCompile(`<br[^>]*>`)
	unescaped = brRegex.ReplaceAllString(unescaped, "\n")

	// Remove other HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	unescaped = tagRegex.ReplaceAllString(unescaped, "")

	// Clean up extra whitespace
	unescaped = strings.TrimSpace(unescaped)
	spaceRegex := regexp.MustCompile(`\n{3,}`)
	unescaped = spaceRegex.ReplaceAllString(unescaped, "\n\n")

	result.WriteString(unescaped)
	return result.String()
}

// Helper function to convert int32 to pointer
func Int32Ptr(i int32) *int32 {
	return &i
}
