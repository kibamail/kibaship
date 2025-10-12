package e2e

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Infrastructure Health Check", func() {
	Context("Operator and Valkey", func() {
		It("should have operator pod running successfully", func() {
			By("verifying operator pod is running")
			Eventually(func() error {
				client, err := getKubernetesClient()
				if err != nil {
					return fmt.Errorf("failed to create kubernetes client: %v", err)
				}

				ctx := context.Background()
				pods, err := client.CoreV1().Pods("kibaship").List(ctx, metav1.ListOptions{
					LabelSelector: "control-plane=controller-manager",
				})
				if err != nil {
					return fmt.Errorf("failed to list operator pods: %v", err)
				}

				if len(pods.Items) == 0 {
					return fmt.Errorf("no operator pods found with label control-plane=controller-manager")
				}

				pod := pods.Items[0]
				if pod.Status.Phase != corev1.PodRunning {
					return fmt.Errorf("operator pod %s is not running, phase: %s", pod.Name, pod.Status.Phase)
				}

				// Also check that all containers are ready
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if !containerStatus.Ready {
						return fmt.Errorf("operator pod %s container %s is not ready", pod.Name, containerStatus.Name)
					}
				}

				return nil
			}, "2m", "10s").Should(Succeed(), "Operator pod should be running")
		})

	})
})
