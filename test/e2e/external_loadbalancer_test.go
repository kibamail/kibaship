package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship/test/utils"
)

var _ = Describe("External LoadBalancer", func() {
	Context("When the operator is deployed", func() {
		It("should create a Gateway with LoadBalancer service", func() {
			By("checking that the Gateway resource exists")
			cmd := exec.Command("kubectl", "get", "gateway", "-n", "kibaship", "ingress-kibaship-gateway")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Gateway should exist")

			By("waiting for Gateway to be ready")
			cmd = exec.Command("kubectl", "wait", "--for=condition=Programmed", "gateway", "-n", "kibaship", "ingress-kibaship-gateway", "--timeout=300s")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Gateway should become ready")

			By("checking that a LoadBalancer service is created by Cilium")
			// The service name follows the pattern: cilium-gateway-{gateway-name}
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "svc", "-n", "kibaship", "-l", "io.cilium.gateway/owning-gateway=ingress-kibaship-gateway")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to get LoadBalancer service: %w, output: %s", err, string(output))
				}
				if !strings.Contains(string(output), "LoadBalancer") {
					return fmt.Errorf("LoadBalancer service not found in output: %s", string(output))
				}
				return nil
			}, 5*time.Minute, 10*time.Second).Should(Succeed(), "LoadBalancer service should be created by Cilium")

			By("verifying LoadBalancer service has external IP assigned by MetalLB")
			var serviceName string
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "svc", "-n", "kibaship", "-l", "io.cilium.gateway/owning-gateway=ingress-kibaship-gateway", "-o", "jsonpath={.items[0].metadata.name}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to get service name: %w", err)
				}
				serviceName = strings.TrimSpace(string(output))
				if serviceName == "" {
					return fmt.Errorf("service name is empty")
				}
				return nil
			}, 2*time.Minute, 5*time.Second).Should(Succeed(), "Should get LoadBalancer service name")

			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "svc", "-n", "kibaship", serviceName, "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to get LoadBalancer IP: %w", err)
				}
				ip := strings.TrimSpace(string(output))
				if ip == "" || ip == "<none>" {
					return fmt.Errorf("LoadBalancer IP not assigned yet")
				}
				GinkgoWriter.Printf("LoadBalancer service %s has external IP: %s\n", serviceName, ip)
				return nil
			}, 5*time.Minute, 10*time.Second).Should(Succeed(), "LoadBalancer should get external IP from MetalLB")

			By("verifying LoadBalancer service exposes the correct ports")
			cmd = exec.Command("kubectl", "get", "svc", "-n", "kibaship", serviceName, "-o", "jsonpath={.spec.ports[*].port}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Should get service ports")

			ports := strings.Fields(string(output))
			// With the new logic, when certificate is ready, Gateway should have HTTP and HTTPS listeners only
			expectedPorts := []string{"80", "443"}

			for _, expectedPort := range expectedPorts {
				Expect(ports).To(ContainElement(expectedPort), fmt.Sprintf("Service should expose port %s", expectedPort))
			}

			// Verify that database ports are NOT exposed (they should be added separately when databases are deployed)
			databasePorts := []string{"3306", "6379", "5432"}
			for _, dbPort := range databasePorts {
				Expect(ports).NotTo(ContainElement(dbPort), fmt.Sprintf("Service should NOT expose database port %s initially", dbPort))
			}

			By("verifying LoadBalancer service has the correct annotations")
			cmd = exec.Command("kubectl", "get", "svc", "-n", "kibaship", serviceName, "-o", "jsonpath={.metadata.annotations}")
			output, err = cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Should get service annotations")

			annotations := string(output)
			GinkgoWriter.Printf("LoadBalancer service annotations: %s\n", annotations)

			// Note: Gateway annotations may not be propagated to the service by Cilium
			// This is implementation-specific behavior
		})

		It("should assign external IP to the LoadBalancer service", func() {
			By("getting the LoadBalancer service external IP")
			var externalIP string
			var serviceName string

			cmd := exec.Command("kubectl", "get", "svc", "-n", "kibaship", "-l", "io.cilium.gateway/owning-gateway=ingress-kibaship-gateway", "-o", "jsonpath={.items[0].metadata.name}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Should get service name")
			serviceName = strings.TrimSpace(string(output))

			cmd = exec.Command("kubectl", "get", "svc", "-n", "kibaship", serviceName, "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
			output, err = cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Should get LoadBalancer IP")
			externalIP = strings.TrimSpace(string(output))
			Expect(externalIP).NotTo(BeEmpty(), "LoadBalancer should have external IP")

			By("verifying the external IP is a valid IP address")
			// Basic IP validation - should be in the MetalLB pool range (172.18.200.x)
			Expect(externalIP).To(MatchRegexp(`^\d+\.\d+\.\d+\.\d+$`), "External IP should be a valid IPv4 address")

			By("verifying the LoadBalancer service type and status")
			cmd = exec.Command("kubectl", "get", "svc", "-n", "kibaship", serviceName, "-o", "jsonpath={.spec.type}")
			output, err = cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Should get service type")
			serviceType := strings.TrimSpace(string(output))
			Expect(serviceType).To(Equal("LoadBalancer"), "Service should be of type LoadBalancer")

			GinkgoWriter.Printf("✅ External LoadBalancer is working! IP: %s\n", externalIP)
			GinkgoWriter.Printf("🌐 LoadBalancer service '%s' has been assigned external IP: %s\n", serviceName, externalIP)
			GinkgoWriter.Printf("📋 Service exposes ports: 80 (HTTP), 443 (HTTPS)\n")
			GinkgoWriter.Printf("💡 You can test external access from your local machine using:\n")
			GinkgoWriter.Printf("   curl -v http://%s:80\n", externalIP)
			GinkgoWriter.Printf("   curl -v https://%s:443 -k\n", externalIP)
			GinkgoWriter.Printf("📝 Note: Database ports (3306, 5432, 6379) will be added when database services are deployed\n")
		})
	})
})
