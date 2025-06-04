/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	resource "k8s.io/apimachinery/pkg/api/resource"

	ravendbv1alpha1 "ravendb-operator/api/v1alpha1"
)

// RavenDBClusterReconciler reconciles a RavenDBCluster object
type RavenDBClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ravendb.ravendb.io,resources=ravendbclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ravendb.ravendb.io,resources=ravendbclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ravendb.ravendb.io,resources=ravendbclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

func (r *RavenDBClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var instance ravendbv1alpha1.RavenDBCluster
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var nodes []ravendbv1alpha1.RavenDBNodeStatus
	for _, node := range instance.Spec.Nodes {

		err := r.ensureNodeStatefulSet(ctx, &instance, node)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.ensureNodeService(ctx, &instance, node)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.ensureNodeIngress(ctx, &instance, node)
		if err != nil {
			return ctrl.Result{}, err
		}

		nodes = append(nodes, ravendbv1alpha1.RavenDBNodeStatus{
			Name:   node.Name,
			Status: "Created",
		})
	}

	instance.Status.Nodes = nodes
	instance.Status.Phase = "Deploying"
	instance.Status.Message = fmt.Sprintf("Ensured desired state for %d RavenDB nodes", len(nodes))
	if err := r.Status().Update(ctx, &instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

func pointerPathType(pt networkingv1.PathType) *networkingv1.PathType {
	return &pt
}

func buildCommonEnvVars(instance *ravendbv1alpha1.RavenDBCluster, node ravendbv1alpha1.RavenDBNode) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: "RAVEN_Setup_Mode", Value: instance.Spec.Mode},
		{Name: "RAVEN_License", Value: instance.Spec.License},
		{Name: "RAVEN_License_Eula_Accepted", Value: "true"},
		{Name: "RAVEN_ServerUrl", Value: instance.Spec.ServerUrl},
		{Name: "RAVEN_ServerUrl_Tcp", Value: instance.Spec.ServerUrlTcp},
		{Name: "RAVEN_PublicServerUrl", Value: node.PublicServerUrl},
		{Name: "RAVEN_PublicServerUrl_Tcp", Value: node.PublicServerUrlTcp},
	}
	return envVars
}

func buildInsecureEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "RAVEN_Security_UnsecuredAccessAllowed", Value: "PublicNetwork"},
	}
}

func buildSecureEnvVars(instance *ravendbv1alpha1.RavenDBCluster) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: "RAVEN_Security_Certificate_Path", Value: "/ravendb/certs/server.pfx"},
		{Name: "RAVEN_Security_Certificate_Exec_TimeoutInSec", Value: "60"},
		{Name: "RAVEN_Security_Certificate_LetsEncrypt_Email", Value: instance.Spec.Email},
	}

	return envVars
}

func (r *RavenDBClusterReconciler) ensureNodeIngress(ctx context.Context, instance *ravendbv1alpha1.RavenDBCluster, node ravendbv1alpha1.RavenDBNode) error {
	ingressName := fmt.Sprintf("ravendb-%s", node.Name)

	var existing networkingv1.Ingress
	err := r.Get(ctx, types.NamespacedName{Name: ingressName, Namespace: instance.Namespace}, &existing)
	if err == nil {
		return nil // Already exists
	}
	if !kerrors.IsNotFound(err) {
		return err
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       "ravendb",
		"app.kubernetes.io/managed-by": "ravendb-operator",
		"app.kubernetes.io/instance":   instance.Name,
	}

	hostBase := fmt.Sprintf("%s.%s", strings.ToLower(node.Name), instance.Spec.Domain)
	tcpHost := fmt.Sprintf("%s-tcp.%s", strings.ToLower(node.Name), instance.Spec.Domain)

	var backendProtocol string
	var webPort int32
	annotations := map[string]string{}

	switch instance.Spec.Mode {
	case "None":
		backendProtocol = "HTTP"
		webPort = 8080

	case "LetsEncrypt":
		backendProtocol = "HTTPS"
		webPort = 443

		annotations["nginx.ingress.kubernetes.io/ssl-passthrough"] = "true"
		annotations["ingress.kubernetes.io/ssl-passthrough"] = "true"

	default:
		return fmt.Errorf("unsupported mode: %s", instance.Spec.Mode)
	}

	annotations["nginx.ingress.kubernetes.io/backend-protocol"] = backendProtocol
	annotations["ingress.kubernetes.io/backend-protocol"] = backendProtocol

	ing := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ingressName,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr.To(instance.Spec.IngressClassName),
			Rules: []networkingv1.IngressRule{
				{
					Host: hostBase,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pointerPathType(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("ravendb-%s", node.Name),
											Port: networkingv1.ServiceBackendPort{Number: webPort},
										},
									},
								},
							},
						},
					},
				},
				{
					Host: tcpHost,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: pointerPathType(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("ravendb-%s", node.Name),
											Port: networkingv1.ServiceBackendPort{Number: 38888},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return r.Create(ctx, &ing)
}

func (r *RavenDBClusterReconciler) ensureNodeService(ctx context.Context, instance *ravendbv1alpha1.RavenDBCluster, node ravendbv1alpha1.RavenDBNode) error {
	svcName := fmt.Sprintf("ravendb-%s", node.Name)

	var existing corev1.Service
	err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: instance.Namespace}, &existing)
	if err == nil {
		return nil // Already exists
	}
	if !kerrors.IsNotFound(err) {
		return err
	}

	var webPort corev1.ServicePort

	switch instance.Spec.Mode {
	case "None":
		webPort = corev1.ServicePort{Name: "http", Port: 8080}
	case "LetsEncrypt":
		webPort = corev1.ServicePort{Name: "https", Port: 443}
	default:
		return fmt.Errorf("unsupported mode: %s", instance.Spec.Mode)
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "ravendb",
				"app.kubernetes.io/managed-by": "ravendb-operator",
				"app.kubernetes.io/instance":   instance.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				webPort,
				{Name: "tcp", Port: 38888, Protocol: corev1.ProtocolTCP},
			},
			Selector: map[string]string{
				"node-tag": node.Name,
			},
		},
	}

	return r.Create(ctx, &svc)
}

func (r *RavenDBClusterReconciler) ensureNodeStatefulSet(ctx context.Context, instance *ravendbv1alpha1.RavenDBCluster, node ravendbv1alpha1.RavenDBNode) error {

	stsName := fmt.Sprintf("ravendb-%s", node.Name)

	var existing appsv1.StatefulSet
	err := r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: instance.Namespace}, &existing)
	if err == nil {
		return nil // Already exists
	}
	if !kerrors.IsNotFound(err) {
		return err
	}

	replicas := int32(1)
	var containerPorts []corev1.ContainerPort
	volumeMounts := []corev1.VolumeMount{
		{Name: "ravendb", MountPath: "/var/lib/ravendb/data"},
	}
	var volumes []corev1.Volume
	annotations := map[string]string{}
	envVars := buildCommonEnvVars(instance, node)

	switch instance.Spec.Mode {
	case "None":
		envVars = append(envVars, buildInsecureEnvVars()...)
		containerPorts = []corev1.ContainerPort{
			{Name: "http", ContainerPort: 8080},
			{Name: "tcp", ContainerPort: 8080},
		}

	case "LetsEncrypt":
		envVars = append(envVars, buildSecureEnvVars(instance)...)
		containerPorts = []corev1.ContainerPort{
			{Name: "https", ContainerPort: 443},
			{Name: "tcp", ContainerPort: 443},
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name: "ravendb-certs", MountPath: "/ravendb/certs",
		})
		volumes = append(volumes, corev1.Volume{
			Name: "ravendb-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: node.CertsSecretRef,
				},
			},
		})

		annotations["ingress.kubernetes.io/ssl-passthrough"] = "true"

	default:
		return fmt.Errorf("unsupported mode: %s", instance.Spec.Mode)
	}

	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app":      "ravendb",
				"node-tag": node.Name,
			},
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.Name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "ravendb", "node-tag": node.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "ravendb", "node-tag": node.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         "ravendb",
							Image:        instance.Spec.Image,
							Env:          envVars,
							Ports:        containerPorts,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ravendb",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(instance.Spec.StorageSize),
							},
						},
					},
				},
			},
		},
	}
	return r.Create(ctx, &sts)

}

// SetupWithManager sets up the controller with the Manager.
func (r *RavenDBClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ravendbv1alpha1.RavenDBCluster{}).
		Complete(r)
}
