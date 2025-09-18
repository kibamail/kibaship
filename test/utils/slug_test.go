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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kibamail/kibaship-operator/pkg/utils"
)

func TestGenerateRandomSlug(t *testing.T) {
	t.Run("generates correct length", func(t *testing.T) {
		slug, err := utils.GenerateRandomSlug()
		require.NoError(t, err)
		assert.Len(t, slug, utils.SlugLength, "Slug should be exactly %d characters", utils.SlugLength)
	})

	t.Run("generates lowercase alphanumeric only", func(t *testing.T) {
		slug, err := utils.GenerateRandomSlug()
		require.NoError(t, err)

		validCharRegex := regexp.MustCompile("^[a-z0-9]+$")
		assert.True(t, validCharRegex.MatchString(slug), "Slug should contain only lowercase letters and numbers: %s", slug)
	})

	t.Run("generates unique slugs", func(t *testing.T) {
		const numSlugs = 100
		slugs := make(map[string]bool)

		for i := 0; i < numSlugs; i++ {
			slug, err := utils.GenerateRandomSlug()
			require.NoError(t, err)

			// Check for duplicates
			assert.False(t, slugs[slug], "Generated duplicate slug: %s", slug)
			slugs[slug] = true
		}

		assert.Len(t, slugs, numSlugs, "Should generate %d unique slugs", numSlugs)
	})

	t.Run("uses all characters from charset", func(t *testing.T) {
		const numTests = 1000
		charUsage := make(map[rune]int)

		// Generate many slugs and track character usage
		for i := 0; i < numTests; i++ {
			slug, err := utils.GenerateRandomSlug()
			require.NoError(t, err)

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
		assert.True(t, foundLetter, "Should use lowercase letters")

		// Verify we use numbers
		foundNumber := false
		for r := '0'; r <= '9'; r++ {
			if charUsage[r] > 0 {
				foundNumber = true
				break
			}
		}
		assert.True(t, foundNumber, "Should use numbers")
	})

	t.Run("distribution is reasonably random", func(t *testing.T) {
		const numSlugs = 1000
		charCounts := make(map[rune]int)

		for i := 0; i < numSlugs; i++ {
			slug, err := utils.GenerateRandomSlug()
			require.NoError(t, err)

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
			assert.True(t, count >= minExpected,
				"Character %c appeared %d times, expected at least %d", char, count, minExpected)
			assert.True(t, count <= maxExpected,
				"Character %c appeared %d times, expected at most %d", char, count, maxExpected)
		}
	})
}

func BenchmarkGenerateRandomSlug(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := utils.GenerateRandomSlug()
		if err != nil {
			b.Fatal(err)
		}
	}
}
