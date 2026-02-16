//go:build e2ev2

package backuprestore

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"github.com/openshift/hypershift/test/e2e/v2/internal"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// VeleroNamespace is the namespace where Velero is deployed
	VeleroNamespace = "openshift-adp"

	// Velero backup phase constants
	BackupPhaseNew                                       = "New"
	BackupPhaseQueued                                    = ""
	BackupPhaseReadyToStart                              = "ReadyToStart"
	BackupPhaseInProgress                                = "InProgress"
	BackupPhaseWaitingForPluginOperations                = "WaitingForPluginOperations"
	BackupPhaseWaitingForPluginOperationsPartiallyFailed = "WaitingForPluginOperationsPartiallyFailed"
	BackupPhaseFinalizing                                = "Finalizing"
	BackupPhaseFinalizingPartiallyFailed                 = "FinalizingPartiallyFailed"
	BackupPhaseCompleted                                 = "Completed"
	BackupPhaseFailed                                    = "Failed"
	BackupPhasePartiallyFailed                           = "PartiallyFailed"
	BackupPhaseDeleting                                  = "Deleting"
)

// EnsureVeleroPodRunning checks if the Velero pod is running in the specified namespace.
func EnsureVeleroPodRunning(testCtx *internal.TestContext, namespace string) error {
	client := testCtx.MgmtClient
	podList := &corev1.PodList{}
	labels := map[string]string{
		"deploy":    "velero",
		"component": "velero",
	}

	if err := client.List(testCtx.Context, podList, crclient.InNamespace(namespace), crclient.MatchingLabels(labels)); err != nil {
		return fmt.Errorf("failed to list Velero pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return fmt.Errorf("no Velero pod found in namespace %s", namespace)
	}

	if len(podList.Items) > 1 {
		return fmt.Errorf("more than one Velero pod found in namespace %s", namespace)
	}

	pod := &podList.Items[0]
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("Velero pod is not running, current phase: %s", pod.Status.Phase)
	}

	return nil
}

// WaitForBackupCompletion waits for a backup to complete using Gomega Eventually.
// If backupName is provided, it waits for that specific backup.
// If backupName is empty, it finds the most recent backup matching the hcName-hcNamespace prefix.
func WaitForBackupCompletion(ctx context.Context, client crclient.Client, oadpNamespace string, backupName string, hcName string, hcNamespace string, timeout time.Duration) error {
	// If no backup name provided, find the most recent backup matching the prefix
	if backupName == "" {
		var err error
		backupName, err = getLatestBackupForHostedCluster(ctx, client, oadpNamespace, hcName, hcNamespace)
		if err != nil {
			return fmt.Errorf("failed to find backup for HostedCluster %s/%s: %w", hcNamespace, hcName, err)
		}
	}

	// Wait for backup to complete using Gomega Eventually
	Eventually(isBackupDone(ctx, client, oadpNamespace, backupName), timeout, 10*time.Second).Should(BeTrue(),
		fmt.Sprintf("backup %s should complete within %v", backupName, timeout))

	// Verify backup completed successfully
	backup, err := getBackup(ctx, client, oadpNamespace, backupName)
	if err != nil {
		return fmt.Errorf("failed to get backup %s: %w", backupName, err)
	}

	phase, _, _ := unstructured.NestedString(backup.Object, "status", "phase")
	if phase != BackupPhaseCompleted {
		failureReason, _, _ := unstructured.NestedString(backup.Object, "status", "failureReason")
		validationErrors, _, _ := unstructured.NestedStringSlice(backup.Object, "status", "validationErrors")
		return fmt.Errorf("backup %s did not complete successfully: phase=%s, failureReason=%s, validationErrors=%v",
			backupName, phase, failureReason, validationErrors)
	}

	return nil
}

// getLatestBackupForHostedCluster finds the most recent backup matching the hcName-hcNamespace prefix
func getLatestBackupForHostedCluster(ctx context.Context, client crclient.Client, oadpNamespace string, hcName string, hcNamespace string) (string, error) {
	backupList := &unstructured.UnstructuredList{}
	backupList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Version: "v1",
		Kind:    "BackupList",
	})
	err := client.List(ctx, backupList, crclient.InNamespace(oadpNamespace))
	if err != nil {
		return "", fmt.Errorf("failed to list backups: %w", err)
	}

	// Filter backups by prefix and collect matching ones
	prefix := fmt.Sprintf("%s-%s-", hcName, hcNamespace)
	var matchingBackups []unstructured.Unstructured
	for _, backup := range backupList.Items {
		if strings.HasPrefix(backup.GetName(), prefix) {
			matchingBackups = append(matchingBackups, backup)
		}
	}

	if len(matchingBackups) == 0 {
		return "", fmt.Errorf("no backups found with prefix %s in namespace %s", prefix, oadpNamespace)
	}

	// Sort by creation timestamp (most recent first)
	sort.Slice(matchingBackups, func(i, j int) bool {
		return matchingBackups[i].GetCreationTimestamp().After(matchingBackups[j].GetCreationTimestamp().Time)
	})

	return matchingBackups[0].GetName(), nil
}

// getBackup retrieves a backup by name
func getBackup(ctx context.Context, client crclient.Client, namespace string, name string) (*unstructured.Unstructured, error) {
	backup := &unstructured.Unstructured{}
	backup.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Version: "v1",
		Kind:    "Backup",
	})
	err := client.Get(ctx, crclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, backup)
	if err != nil {
		return nil, err
	}
	return backup, nil
}

// isBackupDone returns a function that checks if a backup is done
func isBackupDone(ctx context.Context, client crclient.Client, veleroNamespace, name string) func() bool {
	return func() bool {
		backup, err := getBackup(ctx, client, veleroNamespace, name)
		if err != nil {
			return false
		}

		phase, found, err := unstructured.NestedString(backup.Object, "status", "phase")
		if err != nil || !found {
			return false
		}

		// List of phases that indicate the backup is not done
		phasesNotDone := []string{
			BackupPhaseNew,
			BackupPhaseQueued,
			BackupPhaseReadyToStart,
			BackupPhaseInProgress,
			BackupPhaseWaitingForPluginOperations,
			BackupPhaseWaitingForPluginOperationsPartiallyFailed,
			BackupPhaseFinalizing,
			BackupPhaseFinalizingPartiallyFailed,
		}

		for _, notDonePhase := range phasesNotDone {
			if phase == notDonePhase {
				return false
			}
		}
		return true
	}
}
