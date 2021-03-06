package kclient

import (
	"github.com/openshift/odo/pkg/util"
	"testing"

	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ktesting "k8s.io/client-go/testing"
)

// createFakeDeployment creates a fake deployment with the given pod name and labels
func createFakeDeployment(fkclient *Client, fkclientset *FakeClientset, podName string, labels map[string]string) (*appsv1.Deployment, error) {
	fakeUID := types.UID("12345")
	container := GenerateContainer("container1", "image1", true, []string{"tail"}, []string{"-f", "/dev/null"}, []corev1.EnvVar{}, corev1.ResourceRequirements{}, []corev1.ContainerPort{})
	objectMeta := CreateObjectMeta(podName, "default", labels, nil)
	podTemplateSpec := GeneratePodTemplateSpec(objectMeta, []corev1.Container{*container})

	fkclientset.Kubernetes.PrependReactor("create", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
		if podName == "" {
			return true, nil, errors.Errorf("deployment name is empty")
		}
		deployment := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       DeploymentKind,
				APIVersion: DeploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
				UID:  fakeUID,
			},
		}
		return true, &deployment, nil
	})

	deploymentSpec := GenerateDeploymentSpec(*podTemplateSpec)
	createdDeployment, err := fkclient.CreateDeployment(*deploymentSpec)
	if err != nil {
		return nil, err
	}
	return createdDeployment, nil
}

func TestCreateDeployment(t *testing.T) {

	labels := map[string]string{
		"app":       "app",
		"component": "frontend",
	}

	tests := []struct {
		name           string
		deploymentName string
		wantErr        bool
	}{
		{
			name:           "Case: Valid deployment name",
			deploymentName: "pod",
			wantErr:        false,
		},
		{
			name:           "Case: Invalid deployment name",
			deploymentName: "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initialising the fakeclient
			fkclient, fkclientset := FakeNew()
			fkclient.Namespace = "default"

			createdDeployment, err := createFakeDeployment(fkclient, fkclientset, tt.deploymentName, labels)
			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("fkclient.CreateDeployment(pod) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {

				if len(fkclientset.Kubernetes.Actions()) != 1 {
					t.Errorf("expected 1 action in StartDeployment got: %v", fkclientset.Kubernetes.Actions())
				} else {
					if createdDeployment.Name != tt.deploymentName {
						t.Errorf("deployment name does not match the expected name, expected: %s, got %s", tt.deploymentName, createdDeployment.Name)
					}
				}

			}

		})
	}
}

func TestUpdateDeployment(t *testing.T) {

	container := GenerateContainer("container1", "image1", true, []string{"tail"}, []string{"-f", "/dev/null"}, []corev1.EnvVar{}, corev1.ResourceRequirements{}, []corev1.ContainerPort{})

	labels := map[string]string{
		"app":       "app",
		"component": "frontend",
	}

	tests := []struct {
		name           string
		deploymentName string
		wantErr        bool
	}{
		{
			name:           "Case: Valid deployment name",
			deploymentName: "pod",
			wantErr:        false,
		},
		{
			name:           "Case: Invalid deployment name",
			deploymentName: "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initialising the fakeclient
			fkclient, fkclientset := FakeNew()
			fkclient.Namespace = "default"

			objectMeta := CreateObjectMeta(tt.deploymentName, "default", labels, nil)

			podTemplateSpec := GeneratePodTemplateSpec(objectMeta, []corev1.Container{*container})

			fkclientset.Kubernetes.PrependReactor("update", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
				if tt.deploymentName == "" {
					return true, nil, errors.Errorf("deployment name is empty")
				}
				deployment := appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       DeploymentKind,
						APIVersion: DeploymentAPIVersion,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.deploymentName,
					},
				}
				return true, &deployment, nil
			})

			deploymentSpec := GenerateDeploymentSpec(*podTemplateSpec)
			updatedDeployment, err := fkclient.UpdateDeployment(*deploymentSpec)

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("fkclient.UpdateDeployment(pod) unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {

				if len(fkclientset.Kubernetes.Actions()) != 1 {
					t.Errorf("expected 1 action in UpdateDeployment got: %v", fkclientset.Kubernetes.Actions())
				} else {
					if updatedDeployment.Name != tt.deploymentName {
						t.Errorf("deployment name does not match the expected name, expected: %s, got %s", tt.deploymentName, updatedDeployment.Name)
					}
				}

			}

		})
	}
}

func TestDeleteDeployment(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case 1: normal delete with labels",
			args: args{labels: map[string]string{
				"component": "frontend",
			}},
			wantErr: false,
		},
		{
			name:    "case 2: delete with empty labels",
			args:    args{labels: nil},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initialising the fakeclient
			fkclient, fkclientset := FakeNew()
			fkclient.Namespace = "default"

			fkclientset.Kubernetes.PrependReactor("delete-collection", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
				if util.ConvertLabelsToSelector(tt.args.labels) != action.(ktesting.DeleteCollectionAction).GetListRestrictions().Labels.String() {
					return true, nil, errors.Errorf("collection labels are not matching, wanted: %v, got: %v", util.ConvertLabelsToSelector(tt.args.labels), action.(ktesting.DeleteCollectionAction).GetListRestrictions().Labels.String())
				}
				return true, nil, nil
			})

			if err := fkclient.DeleteDeployment(tt.args.labels); (err != nil) != tt.wantErr {
				t.Errorf("DeleteDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
