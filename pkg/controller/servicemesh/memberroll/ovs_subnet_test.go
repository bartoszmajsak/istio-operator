package memberroll

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/maistra/istio-operator/pkg/controller/common/test"
	"github.com/maistra/istio-operator/pkg/controller/common/test/assert"
)

func TestSubnetReconcileNamespaceInMeshDoesNothing(t *testing.T) {
	cl, tracker := test.CreateClient()

	strategy := createAndConfigureSubnetStrategy(cl, t)
	assert.Success(strategy.reconcileNamespaceInMesh(appNamespace), "reconcileNamespaceInMesh", t)
	test.AssertNumberOfWriteActions(t, tracker.Actions(), 0)
}

func TestSubnetRemoveNamespaceFromMeshDoesNothing(t *testing.T) {
	cl, tracker := test.CreateClient()

	strategy := createAndConfigureSubnetStrategy(cl, t)
	assert.Success(strategy.removeNamespaceFromMesh(appNamespace), "removeNamespaceFromMesh", t)
	test.AssertNumberOfWriteActions(t, tracker.Actions(), 0)
}

func createAndConfigureSubnetStrategy(cl client.Client, t *testing.T) *subnetStrategy {
	strategy := &subnetStrategy{}
	return strategy
}
