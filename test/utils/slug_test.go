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

package utils

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship-operator/pkg/utils"
)

var _ = Describe("GenerateRandomSlug", func() {
	It("generates correct length", func() {
		slug, err := utils.GenerateRandomSlug()
		Expect(err).NotTo(HaveOccurred())
		Expect(slug).To(HaveLen(utils.SlugLength))
	})

	It("generates lowercase alphanumeric only", func() {
		slug, err := utils.GenerateRandomSlug()
		Expect(err).NotTo(HaveOccurred())

		validCharRegex := regexp.MustCompile("^[a-z0-9]+$")
		Expect(validCharRegex.MatchString(slug)).To(BeTrue(), "Slug should contain only lowercase letters and numbers: %s", slug)
	})

	It("generates unique slugs", func() {
		const numSlugs = 100
		slugs := make(map[string]bool)

		for i := 0; i < numSlugs; i++ {
			slug, err := utils.GenerateRandomSlug()
			Expect(err).NotTo(HaveOccurred())

			// Check for duplicates
			Expect(slugs[slug]).To(BeFalse(), "Generated duplicate slug: %s", slug)
			slugs[slug] = true
		}

		Expect(slugs).To(HaveLen(numSlugs))
	})

	It("uses all characters from charset", func() {
		const numTests = 1000
		charUsage := make(map[rune]int)

		// Generate many slugs and track character usage
		for i := 0; i < numTests; i++ {
			slug, err := utils.GenerateRandomSlug()
			Expect(err).NotTo(HaveOccurred())

			for _, char := range slug {
				charUsage[char]++
			}
		}

		// Verify we use lowercase letters
		foundLetter := false
		for r := 'a'; r <= 'z'; r++ {
			if charUsage[r] > 0 {
				foundLetter = true
				break
			}
		}
		Expect(foundLetter).To(BeTrue(), "Should use lowercase letters")

		// Verify we use numbers
		foundNumber := false
		for r := '0'; r <= '9'; r++ {
			if charUsage[r] > 0 {
				foundNumber = true
				break
			}
		}
		Expect(foundNumber).To(BeTrue(), "Should use numbers")
	})

	It("distribution is reasonably random", func() {
		const numSlugs = 1000
		charCounts := make(map[rune]int)

		for i := 0; i < numSlugs; i++ {
			slug, err := utils.GenerateRandomSlug()
			Expect(err).NotTo(HaveOccurred())

			for _, char := range slug {
				charCounts[char]++
			}
		}

		totalChars := numSlugs * utils.SlugLength
		expectedAvgPerChar := float64(totalChars) / float64(len(utils.SlugCharset))

		// Each character should appear roughly the expected number of times
		// Allow for 50% variance (random distribution will have some variance)
		minExpected := int(expectedAvgPerChar * 0.5)
		maxExpected := int(expectedAvgPerChar * 1.5)

		for _, char := range utils.SlugCharset {
			count := charCounts[rune(char)]
			Expect(count).To(BeNumerically(">=", minExpected),
				"Character %c appeared %d times, expected at least %d", char, count, minExpected)
			Expect(count).To(BeNumerically("<=", maxExpected),
				"Character %c appeared %d times, expected at most %d", char, count, maxExpected)
		}
	})
})

var _ = Measure("GenerateRandomSlug performance", func(b Benchmarker) {
	b.Time("runtime", func() {
		_, err := utils.GenerateRandomSlug()
		Expect(err).NotTo(HaveOccurred())
	})
}, 100)
