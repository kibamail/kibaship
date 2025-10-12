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
	"context"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/validation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateTestProject creates a test project for use in tests
func CreateTestProject(ctx context.Context, k8sClient client.Client, projectName, projectUUID, projectSlug, workspaceUUID string) (*platformv1alpha1.Project, error) {
	project := &platformv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
			Labels: map[string]string{
				validation.LabelResourceUUID:  projectUUID,
				validation.LabelResourceSlug:  projectSlug,
				validation.LabelWorkspaceUUID: workspaceUUID,
			},
		},
		Spec: platformv1alpha1.ProjectSpec{},
	}
	return project, k8sClient.Create(ctx, project)
}

// CreateTestEnvironment creates a test environment for use in tests
func CreateTestEnvironment(ctx context.Context, k8sClient client.Client, envName, envUUID, envSlug, namespace, projectName, projectUUID string) (*platformv1alpha1.Environment, error) {
	environment := &platformv1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      envName,
			Namespace: namespace,
			Labels: map[string]string{
				validation.LabelResourceUUID: envUUID,
				validation.LabelResourceSlug: envSlug,
				validation.LabelProjectUUID:  projectUUID,
			},
		},
		Spec: platformv1alpha1.EnvironmentSpec{
			ProjectRef: corev1.LocalObjectReference{Name: projectName},
		},
	}
	return environment, k8sClient.Create(ctx, environment)
}

// CreateTestApplication creates a test application for use in tests
func CreateTestApplication(ctx context.Context, k8sClient client.Client, appName, appUUID, appSlug, namespace, environmentName, environmentUUID, projectUUID string, appType platformv1alpha1.ApplicationType) (*platformv1alpha1.Application, error) {
	application := &platformv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				validation.LabelResourceUUID:    appUUID,
				validation.LabelResourceSlug:    appSlug,
				validation.LabelEnvironmentUUID: environmentUUID,
				validation.LabelProjectUUID:     projectUUID,
			},
		},
		Spec: platformv1alpha1.ApplicationSpec{
			EnvironmentRef: corev1.LocalObjectReference{Name: environmentName},
			Type:           appType,
		},
	}
	return application, k8sClient.Create(ctx, application)
}

// CreateTestDeployment creates a test deployment for use in tests
func CreateTestDeployment(ctx context.Context, k8sClient client.Client, deploymentName, deploymentUUID, deploymentSlug, namespace, applicationName, applicationUUID, environmentUUID, projectUUID string) (*platformv1alpha1.Deployment, error) {
	deployment := &platformv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				validation.LabelResourceUUID:    deploymentUUID,
				validation.LabelResourceSlug:    deploymentSlug,
				validation.LabelApplicationUUID: applicationUUID,
				validation.LabelEnvironmentUUID: environmentUUID,
				validation.LabelProjectUUID:     projectUUID,
			},
		},
		Spec: platformv1alpha1.DeploymentSpec{
			ApplicationRef: corev1.LocalObjectReference{Name: applicationName},
		},
	}
	return deployment, k8sClient.Create(ctx, deployment)
}
