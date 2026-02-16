//go:build e2ev2
// +build e2ev2

package backuprestore

import (
	"fmt"

	"github.com/openshift/hypershift/test/e2e/v2/internal"
	corev1 "k8s.io/api/core/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// VeleroNamespace is the namespace where Velero is deployed
	VeleroNamespace = "oadp-operator"
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
