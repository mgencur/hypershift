//go:build e2ev2 && backuprestore

/*
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

package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/oadp"
	"github.com/openshift/hypershift/support/conditions"
	"github.com/openshift/hypershift/test/e2e/util"
	"github.com/openshift/hypershift/test/e2e/v2/backuprestore"
	"github.com/openshift/hypershift/test/e2e/v2/internal"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Context names for backup/restore test phases that can be shared between tests
// and unify naming conventions.
const (
	ContextPreBackupControlPlane   = "PreBackupControlPlane"
	ContextSetupContinual          = "SetupContinual"
	ContextBackup                  = "Backup"
	ContextVerifyContinual         = "VerifyContinual"
	ContextPostBackupControlPlane  = "PostBackupControlPlane"
	ContextRestore                 = "Restore"
	ContextPostRestoreControlPlane = "PostRestoreControlPlane"
	ContextBreakControlPlane       = "BreakControlPlane"
)

var _ = Describe("BackupRestore", Label("backup-restore", "aws"), Ordered, Serial, func() {

	var (
		prober           backuprestore.ProberManager
		testCtx          *internal.TestContext
		backupName       string
		restoreName      string
		scheduleName     string
		excludeWorkloads []string = []string{
			"router", "karpenter", "karpenter-operator", "aws-node-termination-handler",
		}
		expectedConditions []util.Condition
	)

	AfterAll(func() {
		// Safety net for Prober
		if prober != nil {
			err := prober.Stop()
			if err != nil {
				GinkgoLogr.Error(err, "Failed to stop prober")
			}
		}
	})

	BeforeEach(func() {
		testCtx = internal.GetTestContext()
		Expect(testCtx).NotTo(BeNil())
		if err := testCtx.ValidateControlPlaneNamespace(); err != nil {
			AbortSuite(err.Error())
		}
		hostedCluster := testCtx.GetHostedCluster()
		Expect(hostedCluster).NotTo(BeNil(), "HostedCluster should be set up")
		if hostedCluster.Spec.Platform.Type != hyperv1.AWSPlatform {
			Skip("Test is only supported on AWS platform")
		}

		// Ensure Velero pod is running before proceeding with backup/restore tests
		err := backuprestore.EnsureVeleroPodRunning(testCtx)
		if err != nil {
			Fail(fmt.Sprintf("Velero is not running: %v", err))
		}
	})

	Context(ContextPreBackupControlPlane, func() {
		It("should have control plane healthy before backup", func() {
			err := internal.ValidateControlPlaneDeploymentsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			err = internal.ValidateControlPlaneStatefulSetsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			nodePool, err := getNodePool(testCtx)
			Expect(err).NotTo(HaveOccurred())
			Expect(nodePool).NotTo(BeNil())
			for conditionType, conditionStatus := range conditions.ExpectedNodePoolConditions(nodePool) {
				expectedConditions = append(expectedConditions, util.Condition{
					Type:   conditionType,
					Status: metav1.ConditionStatus(conditionStatus),
				})
			}
			internal.ValidateConditions(NewWithT(GinkgoT()), nodePool, expectedConditions)
		})
	})

	// Setup the continual operations
	Context(ContextSetupContinual, func() {
		It("should setup continual operations successfully", func() {
			verifyReconciliationActiveFunction := func() error {
				// Check HostedCluster
				hostedCluster := &hyperv1.HostedCluster{}
				err := testCtx.MgmtClient.Get(testCtx.Context, crclient.ObjectKey{
					Name:      testCtx.ClusterName,
					Namespace: testCtx.ClusterNamespace,
				}, hostedCluster)
				if err != nil {
					return fmt.Errorf("failed to get HostedCluster: %w", err)
				}
				condition := meta.FindStatusCondition(hostedCluster.Status.Conditions, string(hyperv1.ReconciliationActive))
				if condition == nil {
					return fmt.Errorf("HostedCluster ReconciliationActive condition should exist")
				}
				if condition.Status != metav1.ConditionTrue {
					return fmt.Errorf("HostedCluster ReconciliationActive should be always True, but is %s at time %s: %s", condition.Status, condition.LastTransitionTime, condition.Message)
				}

				// Check HostedControlPlane
				hcp := &hyperv1.HostedControlPlane{}
				err = testCtx.MgmtClient.Get(testCtx.Context, crclient.ObjectKey{
					Name:      testCtx.ClusterName,
					Namespace: testCtx.ControlPlaneNamespace,
				}, hcp)
				if err != nil {
					return fmt.Errorf("failed to get HostedControlPlane: %w", err)
				}
				hcpCondition := meta.FindStatusCondition(hcp.Status.Conditions, string(hyperv1.ReconciliationActive))
				if hcpCondition == nil {
					return fmt.Errorf("HostedControlPlane ReconciliationActive condition should exist")
				}
				if hcpCondition.Status != metav1.ConditionTrue {
					return fmt.Errorf("HostedControlPlane ReconciliationActive should be always True, but is %s at time %s: %s", hcpCondition.Status, hcpCondition.LastTransitionTime, hcpCondition.Message)
				}

				// Check NodePool
				nodePool, err := getNodePool(testCtx)
				if err != nil {
					return fmt.Errorf("failed to get NodePool: %w", err)
				}
				for _, npCondition := range nodePool.Status.Conditions {
					if npCondition.Type == hyperv1.NodePoolReconciliationActiveConditionType {
						if npCondition.Status != corev1.ConditionTrue {
							return fmt.Errorf("NodePool ReconciliationActive should be always True, but is %s at time %s: %s", npCondition.Status, npCondition.LastTransitionTime, npCondition.Message)
						}
						break
					}
				}

				// Check MachineDeployment for CAPI paused annotation
				mdList := &capiv1.MachineDeploymentList{}
				err = testCtx.MgmtClient.List(testCtx.Context, mdList, crclient.InNamespace(testCtx.ControlPlaneNamespace))
				if err != nil {
					return fmt.Errorf("failed to list MachineDeployments: %w", err)
				}
				for i := range mdList.Items {
					if _, paused := mdList.Items[i].Annotations[capiv1.PausedAnnotation]; paused {
						return fmt.Errorf("MachineDeployment %s has %s annotation set", mdList.Items[i].Name, capiv1.PausedAnnotation)
					}
				}
				return nil
			}
			prober = backuprestore.NewProberManager(500 * time.Millisecond)
			prober.Spawn(verifyReconciliationActiveFunction)
		})
	})

	Context(ContextBackup, func() {
		It("should create backup and schedule successfully", func() {
			// Create schedule first to test parallel execution of backup and schedule and
			// to speed up the test execution.
			By("Creating schedule")
			scheduleName = oadp.GenerateScheduleName(testCtx.ClusterName, testCtx.ClusterNamespace)
			scheduleOpts := &backuprestore.OADPScheduleOptions{
				Name:            scheduleName,
				Schedule:        "* * * * *", // Every minute
				HCName:          testCtx.ClusterName,
				HCNamespace:     testCtx.ClusterNamespace,
				StorageLocation: testCtx.ClusterName,
			}
			err := backuprestore.RunOADPSchedule(testCtx.Context, GinkgoLogr.WithName("backup-restore"), testCtx.ArtifactDir, scheduleOpts)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				// Prevent creating backups indefinitely through the schedule
				By("Cleaning up schedule")
				if err := backuprestore.DeleteOADPSchedule(testCtx, scheduleName); err != nil {
					GinkgoWriter.Printf("Failed to delete schedule %s during cleanup: %v\n", scheduleName, err)
				}
			})

			By("Creating backup")
			backupName = oadp.GenerateBackupName(
				testCtx.ClusterName,
				testCtx.ClusterNamespace,
			)
			backupOpts := &backuprestore.OADPBackupOptions{
				Name:            backupName,
				HCName:          testCtx.ClusterName,
				HCNamespace:     testCtx.ClusterNamespace,
				StorageLocation: testCtx.ClusterName,
			}
			err = backuprestore.RunOADPBackup(testCtx.Context, GinkgoLogr.WithName("backup-restore"), testCtx.ArtifactDir, backupOpts)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for backup to complete")
			err = backuprestore.WaitForBackupCompletion(testCtx, backupName)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for schedule to have one backup completed")
			err = backuprestore.WaitForScheduleCompletion(testCtx, scheduleName)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// Verify the continual operations
	Context(ContextVerifyContinual, func() {
		It("should verify continual operations completed successfully", func() {
			err := prober.Stop()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextPostBackupControlPlane, func() {
		It("should have control plane healthy after backup", func() {
			err := internal.ValidateControlPlaneDeploymentsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			err = internal.ValidateControlPlaneStatefulSetsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextBreakControlPlane, func() {
		It("should break hosted cluster", func() {
			err := backuprestore.BreakHostedClusterPreservingMachines(testCtx, GinkgoLogr.WithName("cleanup"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextRestore, func() {
		It("should restore from backup successfully", func() {
			By("Creating Restore")
			restoreName = oadp.GenerateRestoreName(testCtx.ClusterName, testCtx.ClusterNamespace)
			restoreOpts := &backuprestore.OADPRestoreOptions{
				Name:        restoreName,
				FromBackup:  backupName,
				HCName:      testCtx.ClusterName,
				HCNamespace: testCtx.ClusterNamespace,
			}
			err := backuprestore.RunOADPRestore(testCtx.Context, GinkgoLogr.WithName("backup-restore"), testCtx.ArtifactDir, restoreOpts)
			Expect(err).NotTo(HaveOccurred())
			By("Waiting for restore to complete")
			err = backuprestore.WaitForRestoreCompletion(testCtx, restoreName)
			Expect(err).NotTo(HaveOccurred())

			By("Fixing OIDC identity provider after restore")
			awsCredsFile := internal.GetEnvVarValue("AWS_GUEST_INFRA_CREDENTIALS_FILE")
			fixOpts := &backuprestore.FixDrOidcIamOptions{
				HCName:       testCtx.ClusterName,
				HCNamespace:  testCtx.ClusterNamespace,
				AWSCredsFile: awsCredsFile,
				Timeout:      backuprestore.OIDCTimeout,
			}
			err = backuprestore.RunFixDrOidcIam(testCtx.Context, GinkgoLogr.WithName("backup-restore"), testCtx.ArtifactDir, fixOpts)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextPostRestoreControlPlane, func() {
		It("should have control plane healthy after restore", func() {
			By("Waiting for control plane statefulsets to be ready")
			err := internal.WaitForControlPlaneStatefulSetsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			By("Waiting for control plane deployments to be ready")
			err = internal.WaitForControlPlaneDeploymentsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			By("Validating NodePool conditions")
			Eventually(func(g Gomega) {
				nodePool, err := getNodePool(testCtx)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(nodePool).NotTo(BeNil())
				internal.ValidateConditions(g, nodePool, expectedConditions)
			}).WithPolling(backuprestore.PollInterval).WithTimeout(backuprestore.OIDCTimeout).Should(Succeed())
		})
	})
})

func getNodePool(testCtx *internal.TestContext) (*hyperv1.NodePool, error) {
	nodePoolList := &hyperv1.NodePoolList{}
	err := testCtx.MgmtClient.List(testCtx.Context, nodePoolList, crclient.InNamespace(testCtx.ClusterNamespace))

	if err != nil {
		return nil, err
	}
	if len(nodePoolList.Items) == 0 {
		return nil, fmt.Errorf("no NodePools found in namespace %s", testCtx.ClusterNamespace)
	}
	for i := range nodePoolList.Items {
		if nodePoolList.Items[i].Spec.ClusterName == testCtx.ClusterName {
			return &nodePoolList.Items[i], nil
		}
	}
	return nil, fmt.Errorf("no NodePool found for cluster %s", testCtx.ClusterName)
}

var _ = Describe("BackupDuringBreak", Label("backup-during-break", "aws"), Ordered, Serial, func() {

	var (
		prober           backuprestore.ProberManager
		testCtx          *internal.TestContext
		excludeWorkloads []string = []string{
			"router", "karpenter", "karpenter-operator", "aws-node-termination-handler",
		}
		expectedConditions []util.Condition
	)

	AfterAll(func() {
		if prober != nil {
			err := prober.Stop()
			if err != nil {
				GinkgoLogr.Error(err, "Failed to stop prober")
			}
		}
	})

	BeforeEach(func() {
		testCtx = internal.GetTestContext()
		Expect(testCtx).NotTo(BeNil())
		if err := testCtx.ValidateControlPlaneNamespace(); err != nil {
			AbortSuite(err.Error())
		}
		hostedCluster := testCtx.GetHostedCluster()
		Expect(hostedCluster).NotTo(BeNil(), "HostedCluster should be set up")
		if hostedCluster.Spec.Platform.Type != hyperv1.AWSPlatform {
			Skip("Test is only supported on AWS platform")
		}

		err := backuprestore.EnsureVeleroPodRunning(testCtx)
		if err != nil {
			Fail(fmt.Sprintf("Velero is not running: %v", err))
		}
	})

	Context(ContextPreBackupControlPlane, func() {
		It("should have control plane healthy before test", func() {
			err := internal.ValidateControlPlaneDeploymentsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			err = internal.ValidateControlPlaneStatefulSetsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			nodePool, err := getNodePool(testCtx)
			Expect(err).NotTo(HaveOccurred())
			Expect(nodePool).NotTo(BeNil())
			for conditionType, conditionStatus := range conditions.ExpectedNodePoolConditions(nodePool) {
				expectedConditions = append(expectedConditions, util.Condition{
					Type:   conditionType,
					Status: metav1.ConditionStatus(conditionStatus),
				})
			}
			internal.ValidateConditions(NewWithT(GinkgoT()), nodePool, expectedConditions)
		})
	})

	Context(ContextSetupContinual, func() {
		It("should setup continual operations with tolerant prober", func() {
			verifyReconciliationActiveFunction := func() error {
				// Check HostedCluster — tolerate NotFound since we break the cluster
				hostedCluster := &hyperv1.HostedCluster{}
				err := testCtx.MgmtClient.Get(testCtx.Context, crclient.ObjectKey{
					Name:      testCtx.ClusterName,
					Namespace: testCtx.ClusterNamespace,
				}, hostedCluster)
				if err != nil {
					if apierrors.IsNotFound(err) {
						GinkgoWriter.Printf("HostedCluster %s/%s not found, skipping check\n", testCtx.ClusterNamespace, testCtx.ClusterName)
					} else {
						return fmt.Errorf("failed to get HostedCluster: %w", err)
					}
				} else {
					condition := meta.FindStatusCondition(hostedCluster.Status.Conditions, string(hyperv1.ReconciliationActive))
					if condition != nil && condition.Status != metav1.ConditionTrue {
						return fmt.Errorf("HostedCluster ReconciliationActive should be True, but is %s at time %s: %s", condition.Status, condition.LastTransitionTime, condition.Message)
					}
				}

				// Check HostedControlPlane — tolerate NotFound
				hcp := &hyperv1.HostedControlPlane{}
				err = testCtx.MgmtClient.Get(testCtx.Context, crclient.ObjectKey{
					Name:      testCtx.ClusterName,
					Namespace: testCtx.ControlPlaneNamespace,
				}, hcp)
				if err != nil {
					if apierrors.IsNotFound(err) {
						GinkgoWriter.Printf("HostedControlPlane %s/%s not found, skipping check\n", testCtx.ControlPlaneNamespace, testCtx.ClusterName)
					} else {
						return fmt.Errorf("failed to get HostedControlPlane: %w", err)
					}
				} else {
					hcpCondition := meta.FindStatusCondition(hcp.Status.Conditions, string(hyperv1.ReconciliationActive))
					if hcpCondition != nil && hcpCondition.Status != metav1.ConditionTrue {
						return fmt.Errorf("HostedControlPlane ReconciliationActive should be True, but is %s at time %s: %s", hcpCondition.Status, hcpCondition.LastTransitionTime, hcpCondition.Message)
					}
				}

				// Check NodePool — tolerate NotFound
				nodePoolList := &hyperv1.NodePoolList{}
				err = testCtx.MgmtClient.List(testCtx.Context, nodePoolList, crclient.InNamespace(testCtx.ClusterNamespace))
				if err != nil {
					if apierrors.IsNotFound(err) {
						GinkgoWriter.Printf("NodePool namespace %s not found, skipping check\n", testCtx.ClusterNamespace)
					} else {
						return fmt.Errorf("failed to list NodePools: %w", err)
					}
				} else {
					for i := range nodePoolList.Items {
						if nodePoolList.Items[i].Spec.ClusterName != testCtx.ClusterName {
							continue
						}
						for _, npCondition := range nodePoolList.Items[i].Status.Conditions {
							if npCondition.Type == hyperv1.NodePoolReconciliationActiveConditionType {
								if npCondition.Status != corev1.ConditionTrue {
									return fmt.Errorf("NodePool %s ReconciliationActive should be True, but is %s at time %s: %s", nodePoolList.Items[i].Name, npCondition.Status, npCondition.LastTransitionTime, npCondition.Message)
								}
								break
							}
						}
					}
				}

				// Check MachineDeployment — tolerate empty list
				mdList := &capiv1.MachineDeploymentList{}
				err = testCtx.MgmtClient.List(testCtx.Context, mdList, crclient.InNamespace(testCtx.ControlPlaneNamespace))
				if err != nil {
					if apierrors.IsNotFound(err) {
						GinkgoWriter.Printf("MachineDeployment namespace %s not found, skipping check\n", testCtx.ControlPlaneNamespace)
					} else {
						return fmt.Errorf("failed to list MachineDeployments: %w", err)
					}
				} else {
					for i := range mdList.Items {
						if _, paused := mdList.Items[i].Annotations[capiv1.PausedAnnotation]; paused {
							return fmt.Errorf("MachineDeployment %s has %s annotation set", mdList.Items[i].Name, capiv1.PausedAnnotation)
						}
					}
				}
				return nil
			}
			prober = backuprestore.NewProberManager(500 * time.Millisecond)
			prober.Spawn(verifyReconciliationActiveFunction)
		})
	})

	Context("BackupBreakLoop", func() {
		It("should survive 10 iterations of backup-during-break", func() {
			const iterations = 10
			for i := 0; i < iterations; i++ {
				iterLabel := fmt.Sprintf("Iteration %d/%d", i+1, iterations)

				// Step 1: Create a good backup and wait for it to complete successfully.
				// This backup will be used for restore after the cluster is broken.
				By(fmt.Sprintf("%s: Creating good backup", iterLabel))
				goodBackupName := oadp.GenerateBackupName(testCtx.ClusterName, testCtx.ClusterNamespace)
				goodBackupOpts := &backuprestore.OADPBackupOptions{
					Name:            goodBackupName,
					HCName:          testCtx.ClusterName,
					HCNamespace:     testCtx.ClusterNamespace,
					StorageLocation: testCtx.ClusterName,
				}
				err := backuprestore.RunOADPBackup(testCtx.Context, GinkgoLogr.WithName("backup-during-break"), testCtx.ArtifactDir, goodBackupOpts)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("%s: Waiting for good backup to complete", iterLabel))
				err = backuprestore.WaitForBackupCompletion(testCtx, goodBackupName)
				Expect(err).NotTo(HaveOccurred())

				// Step 2: Create a second backup that will be corrupted by breaking the cluster.
				By(fmt.Sprintf("%s: Creating disruptive backup", iterLabel))
				disruptiveBackupName := oadp.GenerateBackupName(testCtx.ClusterName, testCtx.ClusterNamespace)
				disruptiveBackupOpts := &backuprestore.OADPBackupOptions{
					Name:            disruptiveBackupName,
					HCName:          testCtx.ClusterName,
					HCNamespace:     testCtx.ClusterNamespace,
					StorageLocation: testCtx.ClusterName,
				}
				err = backuprestore.RunOADPBackup(testCtx.Context, GinkgoLogr.WithName("backup-during-break"), testCtx.ArtifactDir, disruptiveBackupOpts)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("%s: Waiting for disruptive backup to reach InProgress", iterLabel))
				err = backuprestore.WaitForBackupInProgress(testCtx, disruptiveBackupName)
				Expect(err).NotTo(HaveOccurred())

				// Step 3: Break the cluster while the disruptive backup is in progress.
				By(fmt.Sprintf("%s: Breaking hosted cluster", iterLabel))
				err = backuprestore.BreakHostedClusterPreservingMachines(testCtx, GinkgoLogr.WithName("backup-during-break"))
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("%s: Waiting for disruptive backup completion or partial failure", iterLabel))
				err = backuprestore.WaitForBackupCompletionOrPartiallyFailed(testCtx, disruptiveBackupName)
				Expect(err).NotTo(HaveOccurred())

				// Step 4: Restore from the good backup (not the corrupted one).
				By(fmt.Sprintf("%s: Restoring from good backup", iterLabel))
				restoreName := oadp.GenerateRestoreName(testCtx.ClusterName, testCtx.ClusterNamespace)
				restoreOpts := &backuprestore.OADPRestoreOptions{
					Name:        restoreName,
					FromBackup:  goodBackupName,
					HCName:      testCtx.ClusterName,
					HCNamespace: testCtx.ClusterNamespace,
				}
				err = backuprestore.RunOADPRestore(testCtx.Context, GinkgoLogr.WithName("backup-during-break"), testCtx.ArtifactDir, restoreOpts)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("%s: Waiting for restore to complete", iterLabel))
				err = backuprestore.WaitForRestoreCompletion(testCtx, restoreName)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("%s: Fixing OIDC identity provider after restore", iterLabel))
				awsCredsFile := internal.GetEnvVarValue("AWS_GUEST_INFRA_CREDENTIALS_FILE")
				fixOpts := &backuprestore.FixDrOidcIamOptions{
					HCName:       testCtx.ClusterName,
					HCNamespace:  testCtx.ClusterNamespace,
					AWSCredsFile: awsCredsFile,
					Timeout:      backuprestore.OIDCTimeout,
				}
				err = backuprestore.RunFixDrOidcIam(testCtx.Context, GinkgoLogr.WithName("backup-during-break"), testCtx.ArtifactDir, fixOpts)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("%s: Waiting for control plane to be healthy", iterLabel))
				err = internal.WaitForControlPlaneStatefulSetsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
				Expect(err).NotTo(HaveOccurred())
				err = internal.WaitForControlPlaneDeploymentsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context(ContextVerifyContinual, func() {
		It("should verify continual operations completed successfully", func() {
			err := prober.Stop()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("PostTestControlPlane", func() {
		It("should have control plane healthy after all iterations", func() {
			err := internal.ValidateControlPlaneDeploymentsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			err = internal.ValidateControlPlaneStatefulSetsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
