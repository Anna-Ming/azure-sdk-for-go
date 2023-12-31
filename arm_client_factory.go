//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.
// DO NOT EDIT.

package main

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

)

// ArmClientFactory is a client factory used to create any client in this module.
// Don't use this type directly, use NewArmClientFactory instead.
type ArmClientFactory struct {
	subscriptionID string
	credential     azcore.TokenCredential
	options        *arm.ClientOptions
}

// NewArmClientFactory creates a new instance of ArmClientFactory with the specified values.
// The parameter values will be propagated to any client created from this factory.
//   - subscriptionID - The Microsoft Azure subscription ID.
//   - credential - used to authorize requests. Usually a credential from azidentity.
//   - options - pass nil to accept the default values.
func NewArmClientFactory(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*ArmClientFactory, error) {
	_, err := arm.NewClient("armresources"+".ArmClientFactory", "v1.1.1", credential, options)
	if err != nil {
		return nil, err
	}
	return &ArmClientFactory{
		subscriptionID: subscriptionID, credential: credential,
		options: options.Clone(),
	}, nil
}



func (c *ArmClientFactory) NewDeploymentsClient() *armresources.DeploymentsClient {
	subClient, _ := armresources.NewDeploymentsClient(c.subscriptionID, c.credential, c.options)
	return subClient
}

func (c *ArmClientFactory) NewResourceGroupsClient() *armresources.ResourceGroupsClient {
	subClient, _ := armresources.NewResourceGroupsClient(c.subscriptionID, c.credential, c.options)
	return subClient
}

func (c *ArmClientFactory) NewDeploymentStatusClient() *DeploymentStatusClient {
	subClient, _ := NewDeploymentStatusClient(c.subscriptionID, c.credential, c.options)
	return subClient
}