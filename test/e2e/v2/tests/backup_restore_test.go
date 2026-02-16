//go:build e2ev2
// +build e2ev2

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
	"time"

	. "github.com/onsi/ginkgo/v2"
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
)

var _ = Describe("BackupRestoreAWS", Label("backup-restore"), Ordered, func() {

	var (
		prober backuprestore.ProberManager
	)

	BeforeEach(func() {
		testCtx := internal.GetTestContext()

		if err := testCtx.ValidateControlPlaneNamespace(); err != nil {
			AbortSuite(err.Error())
		}

		hostedCluster := testCtx.GetHostedCluster()
		Expect(hostedCluster).NotTo(BeNil(), "HostedCluster should be set up")
		if hostedCluster.Spec.Platform.Type != hyperv1.AWSPlatform {
			Skip("BackupRestoreAWS is only supported on AWS platform")
		}
	})

	Context(ContextSetup, func() {
		It("Setup", func() {
		})
	})

	Context(ContextPreBackupControlPlane, func() {
		It("PreBackupControlPlane", func() {
		})
	})

	Context(ContextPreBackupGuest, func() {
		It("PreBackupGuest", func() {
		})
	})

	// Setup the continual operations
	Context(ContextSetupContinual, func() {
		It("SetupContinual", func() {
			prober = backuprestore.NewProberManager()
			prober.Spawn(func() {
				GinkgoWriter.Println("Probing at " + time.Now().Format(time.RFC3339))
				time.Sleep(200 * time.Millisecond)
				// Fail("test error in continual")
			})
		})
	})

	Context(ContextBackup, func() {
		It("Backup", func() {
			GinkgoWriter.Println("Backup")
			time.Sleep(2 * time.Second)
		})
	})

	// Verify the continual operations
	Context(ContextVerifyContinual, func() {
		It("VerifyContinual", func() {
			prober.Stop()
			GinkgoWriter.Println("Verified Continual test at " + time.Now().Format(time.RFC3339))
		})
	})

	Context(ContextPostBackupControlPlane, func() {
		It("PostBackupControlPlane", func() {
		})
	})

	Context(ContextPostBackupGuest, func() {
		It("PostBackupGuest", func() {
		})
	})

	Context(ContextRestore, func() {
		It("Restore", func() {
		})
	})

	Context(ContextPostRestoreControlPlane, func() {
		It("PostRestoreControlPlane", func() {
		})
	})

	Context(ContextPostRestoreGuest, func() {
		It("PostRestoreGuest", func() {
		})
	})
})
