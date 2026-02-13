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

	// . "github.com/onsi/gomega"
	"github.com/openshift/hypershift/test/e2e/v2/backuprestore"
)

var _ = Describe("BackupRestore", Label("backup-restore"), Ordered, func() {

	var (
		prober backuprestore.ProberManager
	)

	Context("Setup", func() {
		It("Setup", func() {
			GinkgoWriter.Println("Setup")
		})
	})

	Context("PreBackupControlPlane", func() {
		It("PreBackupControlPlane", func() {
			GinkgoWriter.Println("PreBackupControlPlane")
		})
	})

	Context("PreBackupGuest", func() {
		It("PreBackupGuest", func() {
			GinkgoWriter.Println("PreBackupGuest")
		})
	})

	// Setup the continual operations
	Context("SetupContinual", func() {
		It("SetupContinual", func() {
			GinkgoWriter.Println("Background operation started")
			prober = backuprestore.NewProberManager()
			prober.Spawn(func() {
				GinkgoWriter.Println("Probing at " + time.Now().Format(time.RFC3339))
				time.Sleep(500 * time.Millisecond)
				// Fail("test error in continual")
			})
			GinkgoWriter.Println("Background operation completed")
		})
	})

	Context("BackupWith", func() {
		It("Backup", func() {
			GinkgoWriter.Println("Backup")
			time.Sleep(2 * time.Second)
		})
	})

	// Verify the continual operations
	Context("VerifyContinual", func() {
		It("VerifyContinual", func() {
			prober.Stop()
			GinkgoWriter.Println("Verified Continual test at " + time.Now().Format(time.RFC3339))
		})
	})

	Context("PostBackupControlPlane", func() {
		It("PostBackupControlPlane", func() {
			GinkgoWriter.Println("PostBackupControlPlane")
		})
	})

	Context("PostBackupGuest", func() {
		It("PostBackupGuest", func() {
			GinkgoWriter.Println("PostBackupGuest")
		})
	})

	Context("RestoreWith", func() {
		It("Restore", func() {
			GinkgoWriter.Println("Restore")
		})
	})

	Context("PostRestoreControlPlane", func() {
		It("PostRestoreControlPlane", func() {
			GinkgoWriter.Println("PostRestoreControlPlane")
		})
	})

	Context("PostRestoreGuest", func() {
		It("PostRestoreGuest", func() {
			GinkgoWriter.Println("PostRestoreGuest")
		})
	})
})
