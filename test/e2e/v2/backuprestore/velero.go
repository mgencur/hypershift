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

	// Velero restore phase constants
	RestorePhaseNew                                       = "New"
	RestorePhaseInProgress                                = "InProgress"
	RestorePhaseWaitingForPluginOperations                = "WaitingForPluginOperations"
	RestorePhaseWaitingForPluginOperationsPartiallyFailed = "WaitingForPluginOperationsPartiallyFailed"
	RestorePhaseFinalizing                                = "Finalizing"
	RestorePhaseFinalizingPartiallyFailed                 = "FinalizingPartiallyFailed"
	RestorePhaseCompleted                                 = "Completed"
	RestorePhaseFailed                                    = "Failed"
	RestorePhasePartiallyFailed                           = "PartiallyFailed"
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

// WaitForBackupCompletion waits for a backup to complete.
// If backupName is provided, it waits for that specific backup.
// If backupName is empty, it finds the most recent backup matching the HostedCluster name/namespace.
func WaitForBackupCompletion(testCtx *internal.TestContext, oadpNamespace string, backupName string) error {
	// If no backup name provided, find the most recent backup matching the prefix
	if backupName == "" {
		var err error
		backupName, err = getLatestBackupForHostedCluster(testCtx.Context, testCtx.MgmtClient, oadpNamespace, testCtx.ClusterName, testCtx.ClusterNamespace)
		if err != nil {
			return fmt.Errorf("failed to find backup for HostedCluster %s/%s: %w", testCtx.ClusterNamespace, testCtx.ClusterName, err)
		}
	}

	Eventually(isBackupInFinalState(testCtx.Context, testCtx.MgmtClient, oadpNamespace, backupName), BackupTimeout, 10*time.Second).Should(BeTrue(),
		fmt.Sprintf("backup %s should complete within %v", backupName, BackupTimeout))

	return ensureBackupSuccessful(testCtx.Context, testCtx.MgmtClient, oadpNamespace, backupName)
}

// ensureBackupSuccessful verifies that a backup completed successfully.
func ensureBackupSuccessful(ctx context.Context, client crclient.Client, oadpNamespace string, backupName string) error {
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

// isBackupInFinalState returns a function that checks if a backup is in a final state. This
// can be both success and failure.
func isBackupInFinalState(ctx context.Context, client crclient.Client, veleroNamespace, name string) func() bool {
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

// WaitForRestoreCompletion waits for a restore to complete.
// If restoreName is provided, it waits for that specific restore.
// If restoreName is empty, it finds the most recent restore matching the HostedCluster name/namespace.
func WaitForRestoreCompletion(testCtx *internal.TestContext, oadpNamespace string, restoreName string) error {
	// If no restore name provided, find the most recent restore matching the prefix
	if restoreName == "" {
		var err error
		restoreName, err = getLatestRestoreForHostedCluster(testCtx.Context, testCtx.MgmtClient, oadpNamespace, testCtx.ClusterName, testCtx.ClusterNamespace)
		if err != nil {
			return fmt.Errorf("failed to find restore for HostedCluster %s/%s: %w", testCtx.ClusterNamespace, testCtx.ClusterName, err)
		}
	}

	Eventually(isRestoreInFinalState(testCtx.Context, testCtx.MgmtClient, oadpNamespace, restoreName), RestoreTimeout, 10*time.Second).Should(BeTrue(),
		fmt.Sprintf("restore %s should complete within %v", restoreName, RestoreTimeout))

	return ensureRestoreSuccessful(testCtx.Context, testCtx.MgmtClient, oadpNamespace, restoreName)
}

// ensureRestoreSuccessful verifies that a restore completed successfully.
func ensureRestoreSuccessful(ctx context.Context, client crclient.Client, oadpNamespace string, restoreName string) error {
	restore, err := getRestore(ctx, client, oadpNamespace, restoreName)
	if err != nil {
		return fmt.Errorf("failed to get restore %s: %w", restoreName, err)
	}

	phase, _, _ := unstructured.NestedString(restore.Object, "status", "phase")
	if phase != RestorePhaseCompleted {
		failureReason, _, _ := unstructured.NestedString(restore.Object, "status", "failureReason")
		validationErrors, _, _ := unstructured.NestedStringSlice(restore.Object, "status", "validationErrors")
		warnings, _, _ := unstructured.NestedInt64(restore.Object, "status", "warnings")
		errors, _, _ := unstructured.NestedInt64(restore.Object, "status", "errors")
		return fmt.Errorf("restore %s did not complete successfully: phase=%s, failureReason=%s, validationErrors=%v, warnings=%d, errors=%d",
			restoreName, phase, failureReason, validationErrors, warnings, errors)
	}

	return nil
}

// getLatestRestoreForHostedCluster finds the most recent restore matching the hcName-hcNamespace prefix
func getLatestRestoreForHostedCluster(ctx context.Context, client crclient.Client, oadpNamespace string, hcName string, hcNamespace string) (string, error) {
	restoreList := &unstructured.UnstructuredList{}
	restoreList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Version: "v1",
		Kind:    "RestoreList",
	})
	err := client.List(ctx, restoreList, crclient.InNamespace(oadpNamespace))
	if err != nil {
		return "", fmt.Errorf("failed to list restores: %w", err)
	}

	// Filter restores by prefix and collect matching ones
	prefix := fmt.Sprintf("%s-%s-", hcName, hcNamespace)
	var matchingRestores []unstructured.Unstructured
	for _, restore := range restoreList.Items {
		if strings.HasPrefix(restore.GetName(), prefix) {
			matchingRestores = append(matchingRestores, restore)
		}
	}

	if len(matchingRestores) == 0 {
		return "", fmt.Errorf("no restores found with prefix %s in namespace %s", prefix, oadpNamespace)
	}

	// Sort by creation timestamp (most recent first)
	sort.Slice(matchingRestores, func(i, j int) bool {
		return matchingRestores[i].GetCreationTimestamp().After(matchingRestores[j].GetCreationTimestamp().Time)
	})

	return matchingRestores[0].GetName(), nil
}

// getRestore retrieves a restore by name
func getRestore(ctx context.Context, client crclient.Client, namespace string, name string) (*unstructured.Unstructured, error) {
	restore := &unstructured.Unstructured{}
	restore.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Version: "v1",
		Kind:    "Restore",
	})
	err := client.Get(ctx, crclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, restore)
	if err != nil {
		return nil, err
	}
	return restore, nil
}

// isRestoreInFinalState returns a function that checks if a restore is in a final state.
// This can be both success and failure.
func isRestoreInFinalState(ctx context.Context, client crclient.Client, veleroNamespace, name string) func() bool {
	return func() bool {
		restore, err := getRestore(ctx, client, veleroNamespace, name)
		if err != nil {
			return false
		}

		phase, found, err := unstructured.NestedString(restore.Object, "status", "phase")
		if err != nil || !found {
			return false
		}

		// List of phases that indicate the restore is not done
		phasesNotDone := []string{
			RestorePhaseNew,
			RestorePhaseInProgress,
			RestorePhaseWaitingForPluginOperations,
			RestorePhaseWaitingForPluginOperationsPartiallyFailed,
			RestorePhaseFinalizing,
			RestorePhaseFinalizingPartiallyFailed,
		}

		for _, notDonePhase := range phasesNotDone {
			if phase == notDonePhase {
				return false
			}
		}
		return true
	}
}
