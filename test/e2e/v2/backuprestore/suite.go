//go:build e2ev2
// +build e2ev2

package backuprestore

import (
	. "github.com/onsi/ginkgo/v2"
)

type Test struct {
	Name string
	Run  func()
}

type ContinualTest struct {
	Name   string
	Setup  func()
	Verify func()
}

type Suite struct {
	Setup                   []Test
	PreBackupControlPlane   []Test
	PreBackupGuest          []Test
	Continual               []ContinualTest
	Backup                  []Test
	PostBackupControlPlane  []Test
	PostBackupGuest         []Test
	Restore                 []Test
	PostRestoreControlPlane []Test
	PostRestoreGuest        []Test
}

type Tests struct {
}

func (s *Suite) RegisterBackupRestoreTest() bool {
	return Describe("BackupRestore", Label("backup-restore"), Ordered, func() {

		Context("Setup", func() {
			for _, operation := range s.Setup {
				It(operation.Name, operation.Run)
			}
		})

		Context("PreBackupControlPlane", func() {
			for _, operation := range s.PreBackupControlPlane {
				It(operation.Name, operation.Run)
			}
		})

		Context("PreBackupGuest", func() {
			for _, operation := range s.PreBackupGuest {
				It(operation.Name, operation.Run)
			}
		})

		// Setup the continual operations
		Context("SetupContinual", func() {
			for _, operation := range s.Continual {
				It(operation.Name, operation.Setup)
			}
		})

		Context("BackupWith", func() {
			for _, operation := range s.Backup {
				It(operation.Name, operation.Run)
			}
		})

		// Verify the continual operations
		Context("VerifyContinual", func() {
			for _, operation := range s.Continual {
				It(operation.Name, operation.Verify)
			}
		})

		Context("PostBackupControlPlane", func() {
			for _, operation := range s.PostBackupControlPlane {
				It(operation.Name, operation.Run)
			}
		})

		Context("PostBackupGuest", func() {
			for _, operation := range s.PostBackupGuest {
				It(operation.Name, operation.Run)
			}
		})

		Context("RestoreWith", func() {
			for _, operation := range s.Restore {
				It(operation.Name, operation.Run)
			}
		})

		Context("PostRestoreControlPlane", func() {
			for _, operation := range s.PostRestoreControlPlane {
				It(operation.Name, operation.Run)
			}
		})

		Context("PostRestoreGuest", func() {
			for _, operation := range s.PostRestoreGuest {
				It(operation.Name, operation.Run)
			}
		})
	})
}
