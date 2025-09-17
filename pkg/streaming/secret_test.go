/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package streaming

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("SecretManager", func() {
	var (
		k8sClient *mockKubernetesClient
		config    *Config
		manager   SecretManager
	)

	BeforeEach(func() {
		k8sClient = &mockKubernetesClient{}
		config = &Config{
			Namespace:        "test-namespace",
			ValkeySecretName: "test-secret",
		}
		manager = NewSecretManager(k8sClient, config)
	})

	Describe("GetValkeyPassword", func() {
		Context("when password retrieval is successful", func() {
			It("should return the password", func() {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						"password": []byte("test-password"),
					},
				}
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*corev1.Secret)
					*obj = *secret
				})

				password, err := manager.GetValkeyPassword(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(password).To(Equal("test-password"))

				k8sClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when secret is not found", func() {
			It("should return an error", func() {
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("secret not found"))

				password, err := manager.GetValkeyPassword(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to get Valkey secret"))
				Expect(password).To(BeEmpty())

				k8sClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when secret has multiple fields", func() {
			It("should return an error", func() {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						"field1": []byte("value1"),
						"field2": []byte("value2"),
					},
				}
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*corev1.Secret)
					*obj = *secret
				})

				password, err := manager.GetValkeyPassword(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("expected exactly one field in secret, got 2 fields"))
				Expect(password).To(BeEmpty())

				k8sClient.AssertExpectations(GinkgoT())
			})
		})

		Context("when password is empty", func() {
			It("should return an error", func() {
				secret := &corev1.Secret{
					Data: map[string][]byte{
						"password": []byte(""),
					},
				}
				k8sClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*corev1.Secret)
					*obj = *secret
				})

				password, err := manager.GetValkeyPassword(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("secret contains empty password"))
				Expect(password).To(BeEmpty())

				k8sClient.AssertExpectations(GinkgoT())
			})
		})
	})
})
