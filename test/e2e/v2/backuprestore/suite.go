//go:build e2ev2
// +build e2ev2

package backuprestore

import (
	"context"
	"time"

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
	doneCh chan struct{}
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

	// Initialize the done channel for each continual operation
	for i := range s.Tests.Continual {
		s.Tests.Continual[i].doneCh = make(chan struct{})
	}

	// Setup the continual operations
	Context("SetupContinual", func() {
		for _, operation := range s.Tests.Continual {
			It(operation.Name, func() {
				GinkgoWriter.Println("Setup Continual started")
				go func() {
					defer GinkgoRecover()
					defer close(operation.doneCh) // Close channel when goroutine completes
					operation.Setup()
				}()
				GinkgoWriter.Println("Setup Continual completed")
			})
		}
	})

	Context("BackupWith", func() {
		for _, operation := range s.Tests.Backup {
			It(operation.Name, func() {
				operation.Run()
			})
		}
	})

	Context("VerifyContinual", func() {
		for _, operation := range s.Tests.Continual {
			It(operation.Name, func() {
				// Wait for the continual test goroutine to complete
				if operation.doneCh != nil {
					select {
					case <-operation.doneCh:
						GinkgoWriter.Println("Successfully waited for Continual completed")
					case <-time.After(30 * time.Second):
						GinkgoWriter.Println("Timeout waiting for Continual test to be completed")
						Fail("Timeout waiting for Continual test to be completed")
					}
				} else {
					GinkgoWriter.Println("Warning: doneCh channel was not initialized")
				}
			})
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
