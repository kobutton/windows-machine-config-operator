package e2e

import (
	"context"
	"testing"

	mapi "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/windows-machine-config-operator/pkg/secrets"
	"github.com/openshift/windows-machine-config-operator/test/e2e/clusterinfo"
)

func deletionTestSuite(t *testing.T) {
	t.Run("Deletion", func(t *testing.T) { testWindowsNodeDeletion(t) })
}

// testWindowsNodeDeletion tests the Windows node deletion from the cluster.
func testWindowsNodeDeletion(t *testing.T) {
	testCtx, err := NewTestContext()
	require.NoError(t, err)

	// Get all the Machines created by the e2e tests
	e2eMachineSets, err := testCtx.client.Machine.MachineSets(clusterinfo.MachineAPINamespace).List(context.TODO(),
		meta.ListOptions{LabelSelector: clusterinfo.MachineE2ELabel + "=true"})
	require.NoError(t, err, "error listing MachineSets")
	var windowsMachineSetWithLabel *mapi.MachineSet
	for _, machineSet := range e2eMachineSets.Items {
		if machineSet.Spec.Selector.MatchLabels[clusterinfo.MachineOSIDLabel] == "Windows" {
			windowsMachineSetWithLabel = &machineSet
			break
		}
	}

	require.NotNil(t, windowsMachineSetWithLabel, "could not find MachineSet with Windows label")

	// Scale the Windows MachineSet to 0
	expectedNodeCount := int32(0)
	windowsMachineSetWithLabel.Spec.Replicas = &expectedNodeCount
	_, err = testCtx.client.Machine.MachineSets(clusterinfo.MachineAPINamespace).Update(context.TODO(),
		windowsMachineSetWithLabel, meta.UpdateOptions{})
	require.NoError(t, err, "error updating Windows MachineSet")

	// we are waiting 10 minutes for all windows machines to get deleted.
	err = testCtx.waitForWindowsNodes(expectedNodeCount, true, false, false)
	require.NoError(t, err, "Windows node deletion failed")

	t.Run("BYOH node removal", func(t *testing.T) {
		assert.NoError(t, testCtx.deleteWindowsInstanceConfigMap(),
			"error removing windows-instances ConfigMap")
		// TODO: Remove this skip when node removal is implemented as part of https://issues.redhat.com/browse/WINC-582
		t.Skip("not implemented yet")
		err = testCtx.waitForWindowsNodes(expectedNodeCount, true, false, true)
		require.NoError(t, err, "Windows node deletion failed")
	})

	// Cleanup all the MachineSets created by us.
	for _, machineSet := range e2eMachineSets.Items {
		assert.NoError(t, testCtx.deleteMachineSet(&machineSet), "error deleting MachineSet")
	}
	// Phase is ignored during deletion, in this case we are just waiting for Machines to be deleted.
	_, err = testCtx.waitForWindowsMachines(int(expectedNodeCount), "", true)
	require.NoError(t, err, "Machine controller Windows machine deletion failed")
	_, err = testCtx.waitForWindowsMachines(int(expectedNodeCount), "", false)
	require.NoError(t, err, "ConfigMap controller Windows machine deletion failed")

	// Test if prometheus configuration is updated to have no node entries in the endpoints object
	t.Run("Prometheus configuration", testPrometheus)

	// Cleanup secrets created by us.
	err = testCtx.client.K8s.CoreV1().Secrets("openshift-machine-api").Delete(context.TODO(), "windows-user-data", meta.DeleteOptions{})
	require.NoError(t, err, "could not delete userData secret")

	err = testCtx.client.K8s.CoreV1().Secrets("openshift-windows-machine-config-operator").Delete(context.TODO(), secrets.PrivateKeySecret, meta.DeleteOptions{})
	require.NoError(t, err, "could not delete privateKey secret")

	// Cleanup wmco-test namespace created by us.
	err = testCtx.deleteNamespace(testCtx.workloadNamespace)
	require.NoError(t, err, "could not delete test namespace")
}
