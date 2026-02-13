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
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"

	// . "github.com/onsi/gomega"
	"github.com/openshift/hypershift/test/e2e/v2/backuprestore"
)

func setup() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "Setup",
			Run: func() {
				GinkgoWriter.Println("Setup")
			},
		},
	}
}

func preBackupControlPlaneTests() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "PreBackupControlPlane 1",
			Run: func() {
				GinkgoWriter.Println("ControlPlaneVerification 1")
			},
		},
		{
			Name: "PreBackupControlPlane 2",
			Run: func() {
				GinkgoWriter.Println("ControlPlaneVerification 2")
			},
		},
	}
}

func preBackupGuestTests() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "PreBackupGuest",
			Run: func() {
				GinkgoWriter.Println("GuestVerification")
			},
		},
	}
}

func continualTests() []backuprestore.ContinualTest {
	prober := backuprestore.NewProberManager()

	return []backuprestore.ContinualTest{
		{
			Name: "Continual Test",
			// This setup function is run in a background goroutine before the backup is taken.
			Setup: func() {
				GinkgoWriter.Println("Background operation started")
				prober.Spawn(func() {
					GinkgoWriter.Println("Probing at " + time.Now().Format(time.RFC3339))
					time.Sleep(500 * time.Millisecond)
					// Fail("test error in continual")
				})
				GinkgoWriter.Println("Background operation completed")
			},
			// This verify function is run after the backup is taken.
			Verify: func() {
				prober.Stop()
				GinkgoWriter.Println("Verified Continual test at " + time.Now().Format(time.RFC3339))
			},
		},
	}
}

func backup() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "Backup",
			Run: func() {
				time.Sleep(2 * time.Second)
				GinkgoWriter.Println("BackupOperation")
			},
		},
	}
}

func postBackupControlPlaneTests() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "PostBackupControlPlane",
			Run: func() {
				GinkgoWriter.Println("ControlPlaneVerification")
			},
		},
	}
}

func postBackupGuestTests() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "PostBackupGuest",
			Run: func() {
				GinkgoWriter.Println("GuestVerification")
			},
		},
	}
}

func restore() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "Restore",
			Run: func() {
				GinkgoWriter.Println("RestoreOperation")
			},
		},
	}
}

func postRestoreControlPlaneTests() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "PostRestoreControlPlane",
			Run: func() {
				GinkgoWriter.Println("ControlPlaneVerification")
			},
		},
	}
}

func postRestoreGuestTests() []backuprestore.Test {
	return []backuprestore.Test{
		{
			Name: "PostRestoreGuest",
			Run: func() {
				GinkgoWriter.Println("GuestVerification")
			},
		},
	}
}

var suite = &backuprestore.Suite{
	Tests: backuprestore.Tests{
		Setup:                   setup(),
		PreBackupControlPlane:   preBackupControlPlaneTests(),
		PreBackupGuest:          preBackupGuestTests(),
		Continual:               continualTests(),
		Backup:                  backup(),
		PostBackupControlPlane:  postBackupControlPlaneTests(),
		PostBackupGuest:         postBackupGuestTests(),
		Restore:                 restore(),
		PostRestoreControlPlane: postRestoreControlPlaneTests(),
		PostRestoreGuest:        postRestoreGuestTests(),
	},
}

var _ = Describe("BackupRestore", Label("backup-restore"), Ordered, func() {
	suite.RegisterBackupRestoreTest(context.Background())
})
