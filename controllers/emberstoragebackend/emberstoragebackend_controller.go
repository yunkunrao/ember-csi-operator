/*
Copyright 2023.

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

package emberstoragebackend

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	embercsiv1alpha1 "io/ember-csi-manager/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EmberStorageBackendReconciler reconciles a EmberStorageBackend object
type EmberStorageBackendReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ember-csi.io,resources=emberstoragebackends,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ember-csi.io,resources=emberstoragebackends/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ember-csi.io,resources=emberstoragebackends/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EmberStorageBackend object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
// Reconcile reads that state of the cluster for a EmberStorageBackend object and makes changes based on the state read
// and what is in the EmberStorageBackend.Spec
func (r *EmberStorageBackendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	glog.V(3).Infof("Reconciling EmberStorageBackend %s/%s\n", req.Namespace, req.Name)

	// Fetch the EmberStorageBackend instance
	instance := &embercsiv1alpha1.EmberStorageBackend{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Warningf("Failed to get %v: %v", req.NamespacedName, err)
		return reconcile.Result{}, err
	}

	if len(instance.Spec.Config.SysFiles.Name) > 0 {
		key := types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.Config.SysFiles.Name}
		secret := &corev1.Secret{}
		err := r.Client.Get(context.TODO(), key, secret)
		if err != nil {
			glog.Warningf("Failed to get secret %s: %s\n", instance.Spec.Config.SysFiles.Name, err)
		}

		// If multiple keys are found only last one is used
		for key, _ := range secret.Data {
			instance.Spec.Config.SysFiles.Key = key
		}
	}

	backend_config_json, err := interfaceToString(instance.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG)
	if err != nil {
		glog.Errorf("Error parsing X_CSI_BACKEND_CONFIG: %v\n", err)
	}
	setJsonKeyIfEmpty(&backend_config_json, "name", req.Name)

	secrets := map[string][]byte{
		"X_CSI_BACKEND_CONFIG": []byte(backend_config_json),
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ember-csi-operator-%s", req.Name),
			Namespace: req.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
		Data: secrets,
		Type: "ember-csi.io/backend-config",
	}
	err = r.Client.Create(context.TODO(), secret)
	if err != nil && !errors.IsAlreadyExists(err) {
		glog.Errorf("Failed to create a new secret %s in %s: %s", secret.Name, secret.Namespace, err)
	}

	backend_config_map := make(map[string]interface{})
	err = json.Unmarshal([]byte(backend_config_json), &backend_config_map)
	if err == nil {
		// Delete clear-text values, these will be stored in the secret
		for k, _ := range backend_config_map {
			backend_config_map[k] = "REDACTED"
		}
		instance.Spec.Config.EnvVars.X_CSI_BACKEND_CONFIG = backend_config_map
	} else {
		glog.Error("Unmarshal of X_CSI_BACKEND_CONFIG failed: ", err)
	}

	ember_config_json, err := interfaceToString(instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG)
	if err != nil {
		glog.Errorf("Error parsing X_CSI_EMBER_CONFIG: %v\n", err)
	}
	setJsonKeyIfEmpty(&ember_config_json, "plugin_name", req.Name)
	ember_config_map := make(map[string]interface{})
	err = json.Unmarshal([]byte(ember_config_json), &ember_config_map)
	if err == nil {
		instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG = ember_config_map
	} else {
		glog.Error("Unmarshal of X_CSI_EMBER_CONFIG failed: ", err)
	}

	persistence_config_json, err := interfaceToString(instance.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG)
	if err != nil {
		glog.Errorf("Error parsing X_CSI_PERSISTENCE_CONFIG: %v\n", err)
	}
	setJsonKeyIfEmpty(&persistence_config_json, "storage", "crd")
	setJsonKeyIfEmpty(&persistence_config_json, "namespace", instance.Namespace)
	persistence_config_map := make(map[string]interface{})
	err = json.Unmarshal([]byte(persistence_config_json), &persistence_config_map)
	if err == nil {
		instance.Spec.Config.EnvVars.X_CSI_PERSISTENCE_CONFIG = persistence_config_map
	} else {
		glog.Error("Unmarshal of X_CSI_PERSISTENCE_CONFIG failed: ", err)
	}

	//instance.Status.Version = version.Version
	err = r.Client.Update(context.TODO(), instance)
	if err != nil {
		glog.Error("EmberStorageBackend instance update failed: ", err)
	}

	// Manage objects created by the operator
	return reconcile.Result{}, r.handleEmberStorageBackendDeployment(instance)
}

// Manage the Objects created by the Operator.
func (r *EmberStorageBackendReconciler) handleEmberStorageBackendDeployment(instance *embercsiv1alpha1.EmberStorageBackend) error {
	glog.V(3).Infof("Reconciling EmberStorageBackend Deployment Objects")
	// Check if the statefuleSet already exists, if not create a new one
	ss := &appsv1.StatefulSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-controller", instance.Name), Namespace: instance.Namespace}, ss)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		ss = r.statefulSetForEmberStorageBackend(instance)
		glog.V(3).Infof("Creating a new StatefulSet %s in %s", ss.Name, ss.Namespace)
		err = r.Client.Create(context.TODO(), ss)
		if err != nil {
			glog.Errorf("Failed to create a new StatefulSet %s in %s: %s", ss.Name, ss.Namespace, err)
			return err
		}
		glog.V(3).Infof("Successfully Created a new StatefulSet %s in %s", ss.Name, ss.Namespace)
	} else if err != nil {
		glog.Error("Failed to get StatefulSet", err)
		return err
	}

	// Check if the daemonSet already exists, if not create a new one
	ds := &appsv1.DaemonSet{}
	var dSNotFound []int
	daemonSetIndex := 1

	// Check whether topology is enabled. We add +1 because
	// of the default daemonset in addition to the topology ones
	if len(instance.Spec.Topologies) > 0 {
		daemonSetIndex = len(instance.Spec.Topologies) + 1
	}

	for i := 0; i < daemonSetIndex; i++ {
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-node-%d", instance.Name, i), Namespace: instance.Namespace}, ds)
		if err != nil && errors.IsNotFound(err) {
			dSNotFound = append(dSNotFound, i)
		}
	}
	if len(dSNotFound) > 0 {
		// Define new DaemonSet(s)
		for _, daemonSetIndex := range dSNotFound {
			glog.Infof("Trying to create Daemonset with index: %d", daemonSetIndex)
			ds = r.daemonSetForEmberStorageBackend(instance, daemonSetIndex)
			glog.V(3).Infof("Creating a new Daemonset %s in %s", ds.Name, ds.Namespace)
			err = r.Client.Create(context.TODO(), ds)
			if err != nil {
				glog.Errorf("Failed to create a new Daemonset %s in %s: %s", ds.Name, ds.Namespace, err)
				return err
			}
			glog.V(3).Infof("Successfully Created a new Daemonset %s in %s", ds.Name, ds.Namespace)
		}
	} else if err != nil {
		glog.Error("failed to get DaemonSet", err)
		return err
	}

	// Check if the storageclass already exists, if not create a new one
	sc := &storagev1.StorageClass{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name), Namespace: sc.Namespace}, sc)
	if err != nil && errors.IsNotFound(err) {
		// Define a new StorageClass
		sc = r.storageClassForEmberStorageBackend(instance)
		glog.V(3).Infof("Creating a new StorageClass %s in %s", sc.Name, sc.Namespace)
		err = r.Client.Create(context.TODO(), sc)
		if err != nil {
			glog.Errorf("Failed to create a new StorageClass %s in %s: %s", sc.Name, sc.Namespace, err)
			return err
		}
		glog.V(3).Infof("Successfully Created a new StorageClass %s in %s", sc.Name, sc.Namespace)
	} else if err != nil {
		glog.Error("failed to get StorageClass", err)
		return err
	}

	snapShotEnabled := true
	X_CSI_EMBER_CONFIG, err := interfaceToString(instance.Spec.Config.EnvVars.X_CSI_EMBER_CONFIG)
	if err == nil {
		if !isFeatureEnabled(X_CSI_EMBER_CONFIG, "snapshot") {
			snapShotEnabled = false
		}
	} else {
		glog.Errorf("Error parsing X_CSI_EMBER_CONFIG: %v\n", err)
	}

	// Check if the volumeSnapshotClass already exists, if not create a new one. Only valid with CSI Spec > 1.0
	if Conf.getCSISpecVersion() >= 1.0 && snapShotEnabled {
		glog.V(3).Info("Trying to create a new volumeSnapshotClass")
		vsc := &snapshotv1.VolumeSnapshotClass{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name), Namespace: vsc.Namespace}, vsc)
		if err != nil && errors.IsNotFound(err) {
			// Define a new VolumeSnapshotClass
			vsc = r.volumeSnapshotClassForEmberStorageBackend(instance)
			glog.V(3).Infof("Creating a new VolumeSnapshotClass %s in %s", GetPluginDomainName(instance.Name), vsc.Namespace)
			err = r.Client.Create(context.TODO(), vsc)
			if err != nil {
				glog.Errorf("Failed to create a new VolumeSnapshotClass %s in %s: %s", GetPluginDomainName(instance.Name), vsc.Namespace, err)
			}
			glog.V(3).Infof("Successfully Created a new VolumeSnapshotClass %s in %s", GetPluginDomainName(instance.Name), vsc.Namespace)
		} else if err != nil {
			glog.Error("Failed to get VolumeSnapshotClass: ", err)
		}
	}

	// Remove the VolumeSnapshotClass and Update the controller and nodes
	if !snapShotEnabled {
		glog.V(3).Info("Info: Request to disable VolumeSnapshotClass")
		vsc := &snapshotv1.VolumeSnapshotClass{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name), Namespace: vsc.Namespace}, vsc)
		if err != nil && !errors.IsNotFound(err) {
			err = r.Client.Delete(context.TODO(), vsc)
			if err != nil {
				glog.Errorf("Failed to remove VolumeSnapshotClass %s in %s: %s", GetPluginDomainName(instance.Name), vsc.Namespace, err)
			}
		} else if err != nil {
			glog.Error("Failed to get VolumeSnapshotClass: ", err)
		}

		// Update the controller and node
	}

	// Only valid for cluster without using a driver registrar, ie k8s >= 1.13 / ocp >= 4.
	if len(Conf.Sidecars[Cluster].ClusterRegistrar) == 0 && len(Conf.Sidecars[Cluster].Registrar) == 0 {
		// Check if the CSIDriver already exists, if not create a new one
		driver := &storagev1beta1.CSIDriver{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: GetPluginDomainName(instance.Name)}, driver)
		if err != nil && errors.IsNotFound(err) {
			driver = r.csiDriverForEmberStorageBackend(instance)
			glog.V(3).Infof("Creating a new CSIDriver %s", driver.Name)
			err = r.Client.Create(context.TODO(), driver)
			if err != nil {
				glog.Errorf("Failed to create a new CSIDriver %s: %s", driver.Name, err)
				return err
			}
			glog.V(3).Infof("Successfully created a new CSIDriver %s", driver.Name)
		} else if err != nil {
			glog.Errorf("Failed to get CSIDriver %s: %s", driver.Name, err)
			return err
		}
	}

	return nil
}
