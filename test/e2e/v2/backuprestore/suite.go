//go:build e2ev2
// +build e2ev2

package backuprestore

import (
	"context"

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
	Tests Tests
}

type Tests struct {
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

func (s *Suite) RegisterBackupRestoreTest(ctx context.Context) {
	Context("Setup", func() {
		for _, operation := range s.Tests.Setup {
			It(operation.Name, operation.Run)
		}
	})

	Context("PreBackupControlPlane", func() {
		for _, operation := range s.Tests.PreBackupControlPlane {
			It(operation.Name, operation.Run)
		}
	})

	Context("PreBackupGuest", func() {
		for _, operation := range s.Tests.PreBackupGuest {
			It(operation.Name, operation.Run)
		}
	})

	// Setup the continual operations
	Context("SetupContinual", func() {
		for _, operation := range s.Tests.Continual {
			It(operation.Name, operation.Setup)
		}
	})

	Context("BackupWith", func() {
		for _, operation := range s.Tests.Backup {
			It(operation.Name, operation.Run)
		}
	})

	// Verify the continual operations
	Context("VerifyContinual", func() {
		for _, operation := range s.Tests.Continual {
			It(operation.Name, operation.Verify)
		}
	})

	Context("PostBackupControlPlane", func() {
		for _, operation := range s.Tests.PostBackupControlPlane {
			It(operation.Name, operation.Run)
		}
	})

	Context("PostBackupGuest", func() {
		for _, operation := range s.Tests.PostBackupGuest {
			It(operation.Name, operation.Run)
		}
	})

	Context("RestoreWith", func() {
		for _, operation := range s.Tests.Restore {
			It(operation.Name, operation.Run)
		}
	})

	Context("PostRestoreControlPlane", func() {
		for _, operation := range s.Tests.PostRestoreControlPlane {
			It(operation.Name, operation.Run)
		}
	})

	Context("PostRestoreGuest", func() {
		for _, operation := range s.Tests.PostRestoreGuest {
			It(operation.Name, operation.Run)
		}
	})
}
