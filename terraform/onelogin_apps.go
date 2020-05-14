package terraform

import (
	"fmt"
	"log"
	"net/http"

	"github.com/onelogin/onelogin-cli/utils"
	"github.com/onelogin/onelogin-go-sdk/pkg/client"
	"github.com/onelogin/onelogin-go-sdk/pkg/models"
)

type OneloginAppsImportable struct {
	ProviderClient *client.APIClient
	AppType        string
}

// Interface requirement to be an Importable. Calls out to remote (onelogin api) and
// creates their Terraform ResourceDefinitions
func (i OneloginAppsImportable) ImportFromRemote() []ResourceDefinition {
	fmt.Println("Collecting Apps from OneLogin...")
	allApps := i.GetAllApps()
	resourceDefinitions := assembleResourceDefinitions(allApps)
	return resourceDefinitions
}

// helper for packing apps into ResourceDefinitions
func assembleResourceDefinitions(allApps []models.App) []ResourceDefinition {
	resourceDefinitions := make([]ResourceDefinition, len(allApps))
	for i, app := range allApps {
		resourceDefinition := ResourceDefinition{Provider: "onelogin"}
		switch *app.AuthMethod {
		case 8:
			resourceDefinition.Type = "onelogin_oidc_apps"
		case 2:
			resourceDefinition.Type = "onelogin_saml_apps"
		default:
			resourceDefinition.Type = "onelogin_apps"
		}
		resourceDefinition.Name = fmt.Sprintf("%s-%d", utils.ToSnakeCase(utils.ReplaceSpecialChar(*app.Name, "")), *app.ID)
		resourceDefinitions[i] = resourceDefinition
	}
	return resourceDefinitions
}

// Makes the HTTP call to the remote to get the apps using the given query parameters
func (i OneloginAppsImportable) GetAllApps() []models.App {
	var (
		resp    *http.Response
		apps    []models.App
		allApps []models.App
		err     error
		next    string
	)

	appTypeQueryMap := map[string]string{
		"onelogin_apps":      "",
		"onelogin_saml_apps": "2",
		"onelogin_oidc_apps": "8",
	}
	requestedAppType := appTypeQueryMap[i.AppType]

	resp, apps, err = i.ProviderClient.Services.AppsV2.GetApps(&models.AppsQuery{
		AuthMethod: requestedAppType,
	})

	for {
		allApps = append(allApps, apps...)
		next = resp.Header.Get("After-Cursor")
		if next == "" || err != nil {
			break
		}
		resp, apps, err = i.ProviderClient.Services.AppsV2.GetApps(&models.AppsQuery{
			AuthMethod: requestedAppType,
			Cursor:     next,
		})
	}
	if err != nil {
		log.Fatal("error retrieving apps ", err)
	}

	return allApps
}