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
	"crypto/rand"
	"math/big"
)

const (
	// SlugLength is the length of generated project slugs
	SlugLength = 8
	// SlugCharset contains allowed characters for slug generation (lowercase alphanumeric)
	SlugCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// GenerateRandomSlug generates a random 8-character lowercase alphanumeric slug
func GenerateRandomSlug() (string, error) {
	slug := make([]byte, SlugLength)
	charsetLength := big.NewInt(int64(len(SlugCharset)))

	for i := 0; i < SlugLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		slug[i] = SlugCharset[randomIndex.Int64()]
	}

	return string(slug), nil
}
