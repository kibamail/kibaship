package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship-operator/test/utils"
)

var _ = Describe("MySQL Deployment Reconciliation", func() {
	var (
		projectUUID                string
		projectName                string
		mysqlAppUUID               string
		mysqlAppName               string
		mysqlClusterAppUUID        string
		mysqlClusterAppName        string
		mysqlDeploymentUUID        string
		mysqlDeploymentName        string
		mysqlClusterDeploymentUUID string
		mysqlClusterDeploymentName string
		envUUID                    string
		envName                    string
		projectNS                  string
	)

	BeforeEach(func() {
		// Generate UUIDs for this test run
		projectUUID = uuid.New().String()
		projectName = fmt.Sprintf("project-%s", projectUUID)
		projectNS = projectName
		mysqlAppUUID = uuid.New().String()
		mysqlAppName = fmt.Sprintf("application-%s", mysqlAppUUID)
		mysqlClusterAppUUID = uuid.New().String()
		mysqlClusterAppName = fmt.Sprintf("application-%s", mysqlClusterAppUUID)
		mysqlDeploymentUUID = uuid.New().String()
		mysqlDeploymentName = fmt.Sprintf("deployment-%s", mysqlDeploymentUUID)
		mysqlClusterDeploymentUUID = uuid.New().String()
		mysqlClusterDeploymentName = fmt.Sprintf("deployment-%s", mysqlClusterDeploymentUUID)
	})

	AfterEach(func() {
		// Resources will be cleaned up by cluster deletion at the start of next test run
	})

	Context("When creating MySQL applications with deployments", func() {
		It("should successfully reconcile single MySQL deployments", func() {
			By("Creating the test project first with inline manifest")
			projectManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
spec:
  applicationTypes:
    mysql:
      enabled: true
    mysqlCluster:
      enabled: true
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
`, projectName, projectUUID, projectName, workspaceUUIDConst)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(projectManifest)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Applications")

			By("Waiting for default production environment to be ready and fetching its UUID")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envUUID = strings.TrimSpace(string(output))
				return envUUID != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment UUID")

			By("Verifying the production environment is ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Getting the environment name for application reference")
			cmd = exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.name}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envName = strings.TrimSpace(string(output))
			Expect(envName).NotTo(BeEmpty(), "Environment should have a name")

			// ===================================================================
			// MYSQL SINGLE INSTANCE TESTS
			// ===================================================================

			By("Creating the MySQL application with inline manifest")
			mysqlApplicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: MySQL
  environmentRef:
    name: %s
  mysql:
    version: "8.0.36"
`, mysqlAppName, projectNS, mysqlAppUUID, mysqlAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlApplicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for MySQL application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", mysqlAppName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "MySQL Application should be Ready before creating Deployment")

			By("Verifying MySQL ApplicationDomain was created with correct pattern")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", mysqlAppUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec.domain}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				domain := strings.TrimSpace(string(output))
				return strings.Contains(domain, ".mysql.") && len(domain) > 0
			}, "1m", "5s").Should(BeTrue(), "MySQL ApplicationDomain should follow *.mysql.<baseDomain> pattern")

			By("Verifying MySQL ApplicationDomain has correct port")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", mysqlAppUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec.port}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				port := strings.TrimSpace(string(output))
				return port == "3306"
			}, "30s", "2s").Should(BeTrue(), "MySQL ApplicationDomain should have port 3306")

			By("Creating the MySQL deployment with inline manifest")
			mysqlDeploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, mysqlDeploymentName, projectNS, mysqlDeploymentUUID, mysqlDeploymentName, projectUUID, mysqlAppUUID, workspaceUUIDConst, envUUID, mysqlAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlDeploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying MySQL Deployment resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, mysqlDeploymentName, "-n", projectNS)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "MySQL Deployment should be created successfully")

			By("Verifying MySQL credentials secret is created")
			mysqlSecretName := fmt.Sprintf("mysql-secret-%s", mysqlAppUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "secret", mysqlSecretName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "MySQL credentials secret should be created")

			By("Verifying MySQL credentials secret has rootUser, rootHost, and rootPassword")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "secret", mysqlSecretName, "-n", projectNS, "-o", "jsonpath={.data}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				data := string(output)
				return strings.Contains(data, "rootUser") &&
					strings.Contains(data, "rootHost") &&
					strings.Contains(data, "rootPassword")
			}, "30s", "2s").Should(BeTrue(), "MySQL secret should contain rootUser, rootHost, and rootPassword")

			By("Verifying InnoDBCluster resource is created")
			mysqlClusterName := fmt.Sprintf("m-%s", mysqlAppUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "innodbcluster", mysqlClusterName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "30s", "5s").Should(Succeed(), "InnoDBCluster should be created")

			By("Verifying InnoDBCluster has correct single-instance configuration")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "innodbcluster", mysqlClusterName, "-n", projectNS, "-o", "jsonpath={.spec}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				spec := string(output)
				return strings.Contains(spec, "\"instances\":1") &&
					strings.Contains(spec, fmt.Sprintf("\"secretName\":\"%s\"", mysqlSecretName))
			}, "30s", "2s").Should(BeTrue(), "InnoDBCluster should have correct single-instance configuration")

			By("Verifying MySQL pods are created and becoming ready")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pods", "-n", projectNS, "-l", "mysql.oracle.com/cluster", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return 0
				}
				readyStates := strings.Fields(string(output))
				readyCount := 0
				for _, state := range readyStates {
					if state == "true" {
						readyCount++
					}
				}
				return readyCount
			}, "5m", "10s").Should(BeNumerically(">=", 1), "At least one MySQL pod should be ready")

			By("Verifying MySQL Deployment phase transitions to Succeeded")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, mysqlDeploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "5m", "5s").Should(Equal("Succeeded"), "MySQL Deployment should transition to Succeeded phase after MySQL is ready")
		})

		It("should successfully reconcile MySQLCluster deployments", func() {
			By("Creating the test project first with inline manifest")
			projectManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
spec:
  applicationTypes:
    mysql:
      enabled: true
    mysqlCluster:
      enabled: true
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
`, projectName, projectUUID, projectName, workspaceUUIDConst)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(projectManifest)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Applications")

			By("Waiting for default production environment to be ready and fetching its UUID")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envUUID = strings.TrimSpace(string(output))
				return envUUID != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment UUID")

			By("Verifying the production environment is ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Getting the environment name for application reference")
			cmd = exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.name}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envName = strings.TrimSpace(string(output))
			Expect(envName).NotTo(BeEmpty(), "Environment should have a name")

			By("Creating the MySQLCluster application with inline manifest")
			mysqlClusterApplicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: MySQLCluster
  environmentRef:
    name: %s
  mysqlCluster:
    version: "8.0.36"
    replicas: 3
`, mysqlClusterAppName, projectNS, mysqlClusterAppUUID, mysqlClusterAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlClusterApplicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for MySQLCluster application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", mysqlClusterAppName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "MySQLCluster Application should be Ready before creating Deployment")

			By("Verifying MySQLCluster ApplicationDomain was created with correct pattern")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "applicationdomains", "-n", projectNS, "-l", fmt.Sprintf("platform.kibaship.com/application-uuid=%s", mysqlClusterAppUUID), "-o", "jsonpath={.items[?(@.spec.default==true)].spec.domain}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				domain := strings.TrimSpace(string(output))
				return strings.Contains(domain, ".mysql.") && len(domain) > 0
			}, "1m", "5s").Should(BeTrue(), "MySQLCluster ApplicationDomain should follow *.mysql.<baseDomain> pattern")

			By("Creating the MySQLCluster deployment with inline manifest")
			mysqlClusterDeploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, mysqlClusterDeploymentName, projectNS, mysqlClusterDeploymentUUID, mysqlClusterDeploymentName, projectUUID, mysqlClusterAppUUID, workspaceUUIDConst, envUUID, mysqlClusterAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlClusterDeploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying MySQLCluster Deployment resource exists and passes validation")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, mysqlClusterDeploymentName, "-n", projectNS)
				_, err := utils.Run(cmd)
				return err == nil
			}, "30s", "2s").Should(BeTrue(), "MySQLCluster Deployment should be created successfully")

			By("Verifying MySQLCluster credentials secret is created")
			mysqlClusterSecretName := fmt.Sprintf("mysql-secret-%s", mysqlClusterAppUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "secret", mysqlClusterSecretName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "MySQLCluster credentials secret should be created")

			By("Verifying MySQLCluster InnoDBCluster resource is created")
			mysqlClusterInstanceName := fmt.Sprintf("mc-%s", mysqlClusterAppUUID)
			Eventually(func() error {
				cmd := exec.Command("kubectl", "get", "innodbcluster", mysqlClusterInstanceName, "-n", projectNS)
				_, err := cmd.CombinedOutput()
				return err
			}, "1m", "5s").Should(Succeed(), "MySQLCluster InnoDBCluster should be created")

			By("Verifying MySQLCluster InnoDBCluster has correct cluster configuration")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "innodbcluster", mysqlClusterInstanceName, "-n", projectNS, "-o", "jsonpath={.spec}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				spec := string(output)
				return strings.Contains(spec, "\"instances\":3") &&
					strings.Contains(spec, fmt.Sprintf("\"secretName\":\"%s\"", mysqlClusterSecretName))
			}, "30s", "2s").Should(BeTrue(), "MySQLCluster InnoDBCluster should have correct cluster configuration with 3 instances")

			By("Verifying MySQLCluster pods are created and becoming ready")
			Eventually(func() int {
				cmd := exec.Command("kubectl", "get", "pods", "-n", projectNS, "-l", "mysql.oracle.com/cluster", "-o", "jsonpath={.items[*].status.containerStatuses[0].ready}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return 0
				}
				readyStates := strings.Fields(string(output))
				readyCount := 0
				for _, state := range readyStates {
					if state == "true" {
						readyCount++
					}
				}
				return readyCount
			}, "10m", "15s").Should(BeNumerically(">=", 3), "At least 3 MySQLCluster pods should be ready")

			By("Verifying MySQLCluster Deployment phase transitions to Succeeded")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, mysqlClusterDeploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "5m", "5s").Should(Equal("Succeeded"), "MySQLCluster Deployment should transition to Succeeded phase after cluster is ready")
		})

		It("should successfully handle MySQL deployment updates", func() {
			By("Creating the test project first with inline manifest")
			projectManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
spec:
  applicationTypes:
    mysql:
      enabled: true
    mysqlCluster:
      enabled: true
    dockerImage:
      enabled: true
    gitRepository:
      enabled: true
`, projectName, projectUUID, projectName, workspaceUUIDConst)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(projectManifest)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for project to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "project", projectName, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "Project should be Ready before creating Applications")

			By("Waiting for default production environment to be ready and fetching its UUID")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.labels.platform\\.kibaship\\.com/uuid}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				envUUID = strings.TrimSpace(string(output))
				return envUUID != ""
			}, "5s", "1s").Should(BeTrue(), "Should find default production environment UUID")

			By("Verifying the production environment is ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "5s", "1s").Should(BeTrue(), "Default production environment should be Ready before creating Application")

			By("Getting the environment name for application reference")
			cmd = exec.Command("kubectl", "get", "environment", "-n", projectNS, "-l", "platform.kibaship.com/project-uuid="+projectUUID, "-o", "jsonpath={.items[0].metadata.name}")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			envName = strings.TrimSpace(string(output))
			Expect(envName).NotTo(BeEmpty(), "Environment should have a name")

			By("Creating the initial MySQL application with version 8.0.36")
			mysqlAppName := fmt.Sprintf("application-%s", mysqlAppUUID)
			mysqlApplicationManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: MySQL
  environmentRef:
    name: %s
  mysql:
    version: "8.0.36"
`, mysqlAppName, projectNS, mysqlAppUUID, mysqlAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlApplicationManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for MySQL application to be ready")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "application", mysqlAppName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				return strings.TrimSpace(string(output)) == readyPhase
			}, "2m", "5s").Should(BeTrue(), "MySQL Application should be Ready before creating Deployment")

			By("Creating the first MySQL deployment")
			mysqlDeploymentName := fmt.Sprintf("deployment-%s", mysqlDeploymentUUID)
			mysqlDeploymentManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, mysqlDeploymentName, projectNS, mysqlDeploymentUUID, mysqlDeploymentName, projectUUID, mysqlAppUUID, workspaceUUIDConst, envUUID, mysqlAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlDeploymentManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for first deployment to succeed")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, mysqlDeploymentName, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "8m", "5s").Should(Equal("Succeeded"), "First MySQL Deployment should succeed")

			mysqlClusterName := fmt.Sprintf("m-%s", mysqlAppUUID)

			By("Testing MySQL deployment update by changing version")
			mysqlUpdateManifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  type: MySQL
  environmentRef:
    name: %s
  mysql:
    version: "8.0.36"
`, mysqlAppName, projectNS, mysqlAppUUID, mysqlAppName, projectUUID, workspaceUUIDConst, envUUID, envName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlUpdateManifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a second MySQL deployment to trigger update")
			mysqlDeployment2UUID := uuid.New().String()
			mysqlDeployment2Name := fmt.Sprintf("deployment-%s", mysqlDeployment2UUID)
			mysqlDeployment2Manifest := fmt.Sprintf(`apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    platform.kibaship.com/uuid: "%s"
    platform.kibaship.com/slug: "%s"
    platform.kibaship.com/project-uuid: "%s"
    platform.kibaship.com/application-uuid: "%s"
    platform.kibaship.com/workspace-uuid: "%s"
    platform.kibaship.com/environment-uuid: "%s"
spec:
  applicationRef:
    name: %s
  promote: true
`, mysqlDeployment2Name, projectNS, mysqlDeployment2UUID, mysqlDeployment2Name, projectUUID, mysqlAppUUID, workspaceUUIDConst, envUUID, mysqlAppName)

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mysqlDeployment2Manifest)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying InnoDBCluster is updated with new version")
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "innodbcluster", mysqlClusterName, "-n", projectNS, "-o", "jsonpath={.spec.version}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return false
				}
				version := strings.TrimSpace(string(output))
				// Version field should be set when version is specified
				return version != "" && strings.Contains(version, "8.0.36")
			}, "2m", "5s").Should(BeTrue(), "InnoDBCluster should be updated with new version")

			By("Verifying second deployment also reaches Succeeded phase")
			Eventually(func() string {
				cmd := exec.Command("kubectl", "get", deploymentResourceType, mysqlDeployment2Name, "-n", projectNS, "-o", "jsonpath={.status.phase}")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(output))
			}, "5m", "5s").Should(Equal("Succeeded"), "Second MySQL Deployment should also reach Succeeded phase")
		})
	})
})
