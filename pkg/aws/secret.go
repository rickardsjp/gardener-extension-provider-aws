// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	awsclient "github.com/gardener/gardener-extension-provider-aws/pkg/aws/client"
)

// GetCredentialsFromSecretRef reads the secret given by the the secret reference and returns the read Credentials
// object.
func GetCredentialsFromSecretRef(ctx context.Context, client client.Client, secretRef corev1.SecretReference, allowDNSKeys bool) (*Credentials, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, client, &secretRef)
	if err != nil {
		return nil, err
	}
	return ReadCredentialsSecret(secret, allowDNSKeys)
}

// ReadCredentialsSecret reads a secret containing credentials.
func ReadCredentialsSecret(secret *corev1.Secret, allowDNSKeys bool) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	var altAccessKeyIDKey, altSecretAccessKeyKey, altRegionKey *string
	if allowDNSKeys {
		altAccessKeyIDKey, altSecretAccessKeyKey, altRegionKey = ptr.To(DNSAccessKeyID), ptr.To(DNSSecretAccessKey), ptr.To(DNSRegion)
	}

	accessKeyID, err := getSecretDataValue(secret, AccessKeyID, altAccessKeyIDKey, true)
	if err != nil {
		return nil, err
	}

	secretAccessKey, err := getSecretDataValue(secret, SecretAccessKey, altSecretAccessKeyKey, true)
	if err != nil {
		return nil, err
	}

	region, _ := getSecretDataValue(secret, Region, altRegionKey, false)

	return &Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	}, nil
}

// NewClientFromSecretRef creates a new Client for the given AWS credentials from given k8s <secretRef> and
// the AWS region <region>.
func NewClientFromSecretRef(ctx context.Context, client client.Client, secretRef corev1.SecretReference, region string) (awsclient.Interface, error) {
	credentials, err := GetCredentialsFromSecretRef(ctx, client, secretRef, false)
	if err != nil {
		return nil, err
	}
	return awsclient.NewClient(string(credentials.AccessKeyID), string(credentials.SecretAccessKey), region)
}

func getSecretDataValue(secret *corev1.Secret, key string, altKey *string, required bool) ([]byte, error) {
	if value, ok := secret.Data[key]; ok {
		return value, nil
	}
	if altKey != nil {
		if value, ok := secret.Data[*altKey]; ok {
			return value, nil
		}
	}
	if required {
		if altKey != nil {
			return nil, fmt.Errorf("missing %q (or %q) field in secret", key, *altKey)
		}
		return nil, fmt.Errorf("missing %q field in secret", key)
	}
	return nil, nil
}
