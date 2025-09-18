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

package validation

import (
	"regexp"

	"github.com/google/uuid"
)

const (
	// LabelResourceUUID is the label key for resource UUID
	LabelResourceUUID = "platform.kibaship.com/uuid"
	// LabelResourceSlug is the label key for resource slug
	LabelResourceSlug = "platform.kibaship.com/slug"
	// LabelWorkspaceUUID is the label key for workspace UUID (for Projects)
	LabelWorkspaceUUID = "platform.kibaship.com/workspace-uuid"
	// LabelProjectUUID is the label key for project UUID (for Applications, Deployments, ApplicationDomains)
	LabelProjectUUID = "platform.kibaship.com/project-uuid"
	// LabelApplicationUUID is the label key for application UUID (for Deployments, ApplicationDomains)
	LabelApplicationUUID = "platform.kibaship.com/application-uuid"
	// LabelDeploymentUUID is the label key for deployment UUID (for ApplicationDomains)
	LabelDeploymentUUID = "platform.kibaship.com/deployment-uuid"

	// AnnotationResourceName is the annotation key for resource display name
	AnnotationResourceName = "platform.kibaship.com/name"
)

// ValidateUUID validates that a string is a valid UUID format
func ValidateUUID(id string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(id)
}

// ValidateSlug validates that a string is a valid slug format
func ValidateSlug(slug string) bool {
	// Slug must be lowercase alphanumeric with hyphens, no leading/trailing hyphens
	slugRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	return slugRegex.MatchString(slug)
}

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}
