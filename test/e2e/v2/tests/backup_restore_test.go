//go:build e2ev2

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

	"github.com/openshift/hypershift/cmd/oadp"
	"github.com/openshift/hypershift/test/e2e/v2/backuprestore"

	. "github.com/onsi/gomega"
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/test/e2e/v2/internal"
)

// Context names for backup/restore test phases
const (
	ContextSetup                   = "Setup"
	ContextPreBackupControlPlane   = "PreBackupControlPlane"
	ContextPreBackupGuest          = "PreBackupGuest"
	ContextSetupContinual          = "SetupContinual"
	ContextBackup                  = "BackupWith"
	ContextVerifyContinual         = "VerifyContinual"
	ContextPostBackupControlPlane  = "PostBackupControlPlane"
	ContextPostBackupGuest         = "PostBackupGuest"
	ContextRestore                 = "RestoreWith"
	ContextPostRestoreControlPlane = "PostRestoreControlPlane"
	ContextPostRestoreGuest        = "PostRestoreGuest"
	ContextBreakHostedCluster      = "BreakHostedCluster"
)

var _ = Describe("BackupRestoreAWS", Label("backup-restore"), Ordered, func() {

	var (
		prober           backuprestore.ProberManager
		testCtx          *internal.TestContext
		backupName       string
		restoreName      string
		excludeWorkloads []string = []string{
			"router", "karpenter", "karpenter-operator", "aws-node-termination-handler",
		}
	)

	BeforeEach(func() {
		testCtx = internal.GetTestContext()
		if err := testCtx.ValidateControlPlaneNamespace(); err != nil {
			AbortSuite(err.Error())
		}
		hostedCluster := testCtx.GetHostedCluster()
		Expect(hostedCluster).NotTo(BeNil(), "HostedCluster should be set up")
		if hostedCluster.Spec.Platform.Type != hyperv1.AWSPlatform {
			Skip("Test is only supported on AWS platform")
		}

		// Ensure Velero pod is running before proceeding with backup/restore tests
		err := backuprestore.EnsureVeleroPodRunning(testCtx, backuprestore.VeleroNamespace)
		if err != nil {
			Fail(fmt.Sprintf("Velero is not running: %v", err))
		}
	})

	Context(ContextSetup, func() {
		It("should complete setup successfully", func() {
		})
	})

	Context(ContextPreBackupControlPlane, func() {
		It("should have control plane healthy before backup", func() {
			err := internal.ValidateControlPlaneDeploymentsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			err = internal.ValidateControlPlaneStatefulSetsReadiness(testCtx, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextPreBackupGuest, func() {
		It("should have guest cluster ready before backup", func() {
			Skip("Skipping due to OCPBUGS-59876")
		})
	})

	// Setup the continual operations
	Context(ContextSetupContinual, func() {
		It("should setup continual operations successfully", func() {
			prober = backuprestore.NewProberManager()
			prober.Spawn(func() {
				GinkgoWriter.Println("Probing at " + time.Now().Format(time.RFC3339))
				time.Sleep(1 * time.Second)
			})
		})
	})

	Context(ContextBackup, func() {
		It("should create backup successfully", func() {
			By("Creating backup")
			backupName = oadp.GenerateBackupName(testCtx.GetHostedCluster().Name, testCtx.GetHostedCluster().Namespace)
			backupOpts := &backuprestore.OADPBackupOptions{
				HCName:          testCtx.GetHostedCluster().Name,
				HCNamespace:     testCtx.GetHostedCluster().Namespace,
				Name:            backupName,
				StorageLocation: testCtx.GetHostedCluster().Name,
			}
			err := backuprestore.RunOADPBackup(testCtx.Context, GinkgoLogr.WithName("backup-restore"), testCtx.ArtifactDir, backupOpts)
			Expect(err).NotTo(HaveOccurred())

			// Wait for backup to complete
			By("Waiting for backup to complete")
			err = backuprestore.WaitForBackupCompletion(
				testCtx,
				backuprestore.VeleroNamespace,
				backupOpts.Name,
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// Verify the continual operations
	Context(ContextVerifyContinual, func() {
		It("should verify continual operations completed successfully", func() {
			prober.Stop()
			GinkgoWriter.Println("Verified Continual test at " + time.Now().Format(time.RFC3339))
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

	Context(ContextPostBackupGuest, func() {
		It("should have guest cluster healthy after backup", func() {
			Skip("Skipping due to OCPBUGS-59876")
		})
	})

	Context(ContextBreakHostedCluster, func() {
		It("should break hosted cluster", func() {
			err := backuprestore.BreakHostedCluster(testCtx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextRestore, func() {
		It("should restore from backup successfully", func() {
			By("Creating Restore")
			restoreName = oadp.GenerateRestoreName(testCtx.GetHostedCluster().Name, testCtx.GetHostedCluster().Namespace)
			restoreOpts := &backuprestore.OADPRestoreOptions{
				Name:        restoreName,
				HCName:      testCtx.GetHostedCluster().Name,
				HCNamespace: testCtx.GetHostedCluster().Namespace,
				FromBackup:  backupName,
			}
			err := backuprestore.RunOADPRestore(testCtx.Context, GinkgoLogr.WithName("backup-restore"), testCtx.ArtifactDir, restoreOpts)
			Expect(err).NotTo(HaveOccurred())

			// Wait for restore to complete
			By("Waiting for restore to complete")
			err = backuprestore.WaitForRestoreCompletion(
				testCtx,
				backuprestore.VeleroNamespace,
				restoreName,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for control plane statefulsets to be ready")
			err = internal.WaitForControlPlaneStatefulSetsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for control plane deployments to be ready")
			err = internal.WaitForControlPlaneDeploymentsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextPostRestoreControlPlane, func() {
		It("should have control plane healthy after restore", func() {
			err := internal.WaitForControlPlaneStatefulSetsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
			err = internal.WaitForControlPlaneDeploymentsReadiness(testCtx, backuprestore.RestoreTimeout, excludeWorkloads)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context(ContextPostRestoreGuest, func() {
		It("should have guest cluster healthy after restore", func() {
			Skip("Skipping due to OCPBUGS-59876")
		})
	})
})
