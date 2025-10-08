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

package controller

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Domain Utils", func() {
	Describe("ValidateDomainFormat", func() {
		DescribeTable("domain validation",
			func(domain string, expectError bool) {
				err := ValidateDomainFormat(domain)
				if expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			},
			Entry("valid domain", "frontend-abc123.myapps.kibaship.com", false),
			Entry("valid subdomain", "app.example.com", false),
			Entry("empty domain", "", true),
			Entry("domain with uppercase", "Frontend-ABC123.myapps.kibaship.com", true),
			Entry("domain starting with hyphen", "-frontend.myapps.kibaship.com", true),
			Entry("domain ending with hyphen", "frontend-.myapps.kibaship.com", true),
			Entry("domain with double dots", "frontend..myapps.kibaship.com", true),
		)
	})

	Describe("GenerateSubdomain", func() {
		DescribeTable("subdomain generation",
			func(appUUID string) {
				subdomain, err := GenerateSubdomain(appUUID)
				Expect(err).NotTo(HaveOccurred())

				// Check that subdomain is not empty
				Expect(subdomain).NotTo(BeEmpty())

				// Check that subdomain follows expected pattern (contains hyphen for random suffix)
				Expect(subdomain).To(ContainSubstring("-"))

				// Check that subdomain is valid DNS format
				Expect(ValidateDomainFormat(subdomain + ".example.com")).To(Succeed())

				// Check length constraints
				Expect(len(subdomain)).To(BeNumerically("<=", 60))

				// Check that it only contains allowed characters
				for _, r := range subdomain {
					Expect((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-').To(BeTrue(),
						"subdomain contains invalid character '%c'", r)
				}
			},
			Entry("frontend app UUID", "123e4567-e89b-12d3-a456-426614174000"),
			Entry("api gateway app UUID", "987fcdeb-51a2-43d7-9876-543210fedcba"),
		)
	})

	Describe("GenerateApplicationDomainName", func() {
		Context("with default domain type", func() {
			It("generates expected domain name", func() {
				appName := "project-myproject-app-frontend"
				result := GenerateApplicationDomainName(appName, "default")
				Expect(result).To(Equal("project-myproject-app-frontend-default"))
			})
		})

		Context("with custom domain type", func() {
			It("generates domain name with random suffix", func() {
				appName := "project-myproject-app-frontend"
				result := GenerateApplicationDomainName(appName, "custom")

				expectedPrefix := appName + "-custom-"
				Expect(result).To(HavePrefix(expectedPrefix))

				// Check that it has a suffix after the prefix
				suffix := strings.TrimPrefix(result, expectedPrefix)
				Expect(suffix).NotTo(BeEmpty())
			})
		})
	})
})
