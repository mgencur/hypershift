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
	// . "github.com/onsi/gomega"
)

// Context names for backup/restore test phases
const (
	contextSetup                   = "Setup"
	contextPreBackupControlPlane   = "PreBackupControlPlane"
	contextPreBackupGuest          = "PreBackupGuest"
	contextSetupContinual          = "SetupContinual"
	contextBackup                  = "BackupWith"
	contextVerifyContinual         = "VerifyContinual"
	contextPostBackupControlPlane  = "PostBackupControlPlane"
	contextPostBackupGuest         = "PostBackupGuest"
	contextRestore                 = "RestoreWith"
	contextPostRestoreControlPlane = "PostRestoreControlPlane"
	contextPostRestoreGuest        = "PostRestoreGuest"
)

var _ = Describe("BackupRestore", Label("backup-restore"), Ordered, func() {

	var (
		prober backuprestore.ProberManager
	)

	Context(contextSetup, func() {
		It("Setup", func() {
		})
	})

	Context(contextPreBackupControlPlane, func() {
		It("PreBackupControlPlane", func() {
		})
	})

	Context(contextPreBackupGuest, func() {
		It("PreBackupGuest", func() {
		})
	})

	// Setup the continual operations
	Context(contextSetupContinual, func() {
		It("SetupContinual", func() {
			prober = backuprestore.NewProberManager()
			prober.Spawn(func() {
				GinkgoWriter.Println("Probing at " + time.Now().Format(time.RFC3339))
				time.Sleep(200 * time.Millisecond)
				// Fail("test error in continual")
			})
		})
	})

	Context(contextBackup, func() {
		It("Backup", func() {
			GinkgoWriter.Println("Backup")
			time.Sleep(2 * time.Second)
		})
	})

	// Verify the continual operations
	Context(contextVerifyContinual, func() {
		It("VerifyContinual", func() {
			prober.Stop()
			GinkgoWriter.Println("Verified Continual test at " + time.Now().Format(time.RFC3339))
		})
	})

	Context(contextPostBackupControlPlane, func() {
		It("PostBackupControlPlane", func() {
		})
	})

	Context(contextPostBackupGuest, func() {
		It("PostBackupGuest", func() {
		})
	})

	Context(contextRestore, func() {
		It("Restore", func() {
		})
	})

	Context(contextPostRestoreControlPlane, func() {
		It("PostRestoreControlPlane", func() {
		})
	})

	Context(contextPostRestoreGuest, func() {
		It("PostRestoreGuest", func() {
		})
	})
})
