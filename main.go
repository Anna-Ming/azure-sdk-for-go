// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var (
	subscriptionID    string
	location          = "westeurope"
	resourceGroupName = "vNextPoc-Anna"
	deploymentName    string
)

var (
	resourcesClientFactory *ArmClientFactory
)

var (
	resourceGroupClient     *armresources.ResourceGroupsClient
	deploymentsClient       *armresources.DeploymentsClient
	deploymentsStatusClient *DeploymentStatusClient
)

func getDeploymentStatus(ctx context.Context, asyncOperation string) (DeploymentStatusResponse, error) {

	resp, err := deploymentsStatusClient.GetDeploymentStatus(ctx, asyncOperation)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func mainCreateDeployment() {
	subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	resourcesClientFactory, err = NewArmClientFactory(subscriptionID, cred, nil)
	if err != nil {
		log.Fatal(err)
	}
	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()
	deploymentsClient = resourcesClientFactory.NewDeploymentsClient()
	deploymentsStatusClient = resourcesClientFactory.NewDeploymentStatusClient()

	resourceGroup, err := createResourceGroup(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("resources group id:", *resourceGroup.ID)

	exist, err := checkExistDeployment(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("deployment is exist:", exist)

	template, err := readJson("day1data/template.json")
	if err != nil {
		log.Fatal(err)
	}
	params, err := readJson("day1data/parameters.json")
	if err != nil {
		log.Fatal(err)
	}
	asyncOperation, err := createDeploymentNoPoll(ctx, template, params)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("asyncOperation: ", asyncOperation)
	log.Println("created deployment: ", deploymentName)

	//check deployment status, using asyncOperation as URL
	re := regexp.MustCompile("(?i)^[^?]+")
	statusURL := re.FindString(asyncOperation)
	log.Println("statusURL is: ", statusURL)
	statusResp, err := getDeploymentStatus(ctx, statusURL)
	if err != nil {
		log.Fatal(err)
	}
	code := statusResp.Error["code"]
	message := statusResp.Error["message"]

	log.Println("statusResp code:", code)
	log.Println("statusResp message:", message)

	if details, ok := statusResp.Error["details"].([]interface{}); ok {
		if len(details) > 0 {
			detailsCode := details[0].(map[string]interface{})["code"]
			fmt.Println(detailsCode)
			detailsMessage := details[0].(map[string]interface{})["message"]
			fmt.Println(detailsMessage)
		}
	}

	log.Println("statusResp:", statusResp)

	validateResult, err := validateDeployment(ctx, template, params)
	if err != nil {
		log.Fatal(err)
	}
	data, _ := json.Marshal(validateResult)
	log.Println("validate deployment:", string(data))

	keepResource := os.Getenv("KEEP_RESOURCE")
	if len(keepResource) == 0 {
		err = cleanup(ctx)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("cleaned up successfully.")
	}
}

func mainGetStatus() {

}

func main() {
	//set deployment name with random number
	deploymentName = fmt.Sprintf("%s_%s", "my_deployment", strconv.Itoa(rand.Intn(1000)))
	//set environment variables
	os.Setenv("AZURE_TENANT_ID", "db42a3c1-b08d-45bc-bc52-9301ef2277c5")
	os.Setenv("AZURE_SUBSCRIPTION_ID", "7fe9165d-336e-4da7-b939-072eb89d9c3a")
	os.Setenv("AZURE_CLIENT_ID", "cc7009d3-adfa-43a1-8d13-f3082884274a")
	os.Setenv("AZURE_CLIENT_SECRET", "u6h8Q~hAanDDqek4ABw6t.tZKbMaR2xlKSN2Zda7")
	os.Setenv("KEEP_RESOURCE", "1")

	fmt.Println("deploymentName is:", deploymentName)
	fmt.Println("AZURE_SUBSCRIPTION_ID:", os.Getenv("AZURE_SUBSCRIPTION_ID"))

	mainCreateDeployment()

}

func readJson(path string) (map[string]interface{}, error) {
	templateFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	template := make(map[string]interface{})
	if err := json.Unmarshal(templateFile, &template); err != nil {
		return nil, err
	}

	return template, nil
}

func checkExistDeployment(ctx context.Context) (bool, error) {

	boolResp, err := deploymentsClient.CheckExistence(ctx, resourceGroupName, deploymentName, nil)
	if err != nil {
		return false, err
	}

	return boolResp.Success, nil
}

func createDeploymentNoPoll(ctx context.Context, template, params map[string]interface{}) (string, error) {

	deploymentPollerResp, err := deploymentsClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		deploymentName,
		armresources.Deployment{
			Properties: &armresources.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       to.Ptr(armresources.DeploymentModeIncremental),
			},
		},
		nil)

	if err != nil {
		return "", fmt.Errorf("cannot create deployment: %v", err)
	}

	resp, err := deploymentPollerResp.Poll(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot get the create deployment respone: %v", err)
	}
	log.Println("resp.Request.URL.Path:", resp.Request.URL.Path)
	log.Println("resp.Request.URL.RawQuery:", resp.Request.URL.RawQuery)

	//Get the initial response of the deployment
	asyncOperation := fmt.Sprintf("%s?%s", resp.Request.URL.Path, resp.Request.URL.RawQuery)

	return asyncOperation, nil
}

func createDeployment(ctx context.Context, template, params map[string]interface{}) (*armresources.DeploymentExtended, error) {

	deploymentPollerResp, err := deploymentsClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		deploymentName,
		armresources.Deployment{
			Properties: &armresources.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       to.Ptr(armresources.DeploymentModeIncremental),
			},
		},
		nil)

	if err != nil {
		return nil, fmt.Errorf("cannot create deployment: %v", err)
	}

	resp, err := deploymentPollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get the create deployment future respone: %v", err)
	}

	return &resp.DeploymentExtended, nil
}

func validateDeployment(ctx context.Context, template, params map[string]interface{}) (*armresources.DeploymentValidateResult, error) {

	pollerResp, err := deploymentsClient.BeginValidate(
		ctx,
		resourceGroupName,
		deploymentName,
		armresources.Deployment{
			Properties: &armresources.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       to.Ptr(armresources.DeploymentModeIncremental),
			},
		},
		nil)

	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.DeploymentValidateResult, nil
}

func createResourceGroup(ctx context.Context) (*armresources.ResourceGroup, error) {

	resourceGroupResp, err := resourceGroupClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		armresources.ResourceGroup{
			Location: to.Ptr(location),
		},
		nil)
	if err != nil {
		return nil, err
	}
	return &resourceGroupResp.ResourceGroup, nil
}

func cleanup(ctx context.Context) error {

	pollerResp, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}
