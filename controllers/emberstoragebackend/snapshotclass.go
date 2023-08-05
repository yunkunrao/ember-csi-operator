package emberstoragebackend

import (
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	embercsiv1alpha1 "io/ember-csi-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// volumeSnapshotClassForEmberStorageBackend returns a EmberStorageBackend VolumeSnapshotClass object
func (r *EmberStorageBackendReconciler) volumeSnapshotClassForEmberStorageBackend(ecsi *embercsiv1alpha1.EmberStorageBackend) *snapshotv1.VolumeSnapshotClass {
	ls := labelsForEmberStorageBackend(ecsi.Name)

	vsc := &snapshotv1.VolumeSnapshotClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "snapshot.storage.k8s.io/v1beta1",
			Kind:       "VolumeSnapshotClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetPluginDomainName(ecsi.Name),
			Namespace: ecsi.Namespace,
			Labels:    ls,
		},
		Driver:         GetPluginDomainName(ecsi.Name),
		DeletionPolicy: "Delete",
	}

	controllerutil.SetControllerReference(ecsi, vsc, r.Scheme)
	return vsc
}
