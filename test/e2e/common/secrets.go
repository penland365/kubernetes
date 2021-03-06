/*
Copyright 2014 The Kubernetes Authors.

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

package common

import (
	"fmt"
	"os"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = framework.KubeDescribe("Secrets", func() {
	f := framework.NewDefaultFramework("secrets")

	It("should be consumable from pods in volume [Conformance]", func() {
		doSecretE2E(f, nil)
	})

	It("should be consumable from pods in volume with defaultMode set [Conformance]", func() {
		defaultMode := int32(0400)
		doSecretE2E(f, &defaultMode)
	})

	It("should be consumable from pods in volume with Mode set in the item [Conformance]", func() {
		name := "secret-test-itemmode-" + string(uuid.NewUUID())
		volumeName := "secret-volume"
		volumeMountPath := "/etc/secret-volume"
		secret := secretForTest(f.Namespace.Name, name)

		By(fmt.Sprintf("Creating secret with name %s", secret.Name))
		var err error
		if secret, err = f.Client.Secrets(f.Namespace.Name).Create(secret); err != nil {
			framework.Failf("unable to create test secret %s: %v", secret.Name, err)
		}

		mode := int32(0400)
		pod := &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: "pod-secrets-" + string(uuid.NewUUID()),
			},
			Spec: api.PodSpec{
				Volumes: []api.Volume{
					{
						Name: volumeName,
						VolumeSource: api.VolumeSource{
							Secret: &api.SecretVolumeSource{
								SecretName: name,
								Items: []api.KeyToPath{
									{
										Key:  "data-1",
										Path: "data-1",
										Mode: &mode,
									},
								},
							},
						},
					},
				},
				Containers: []api.Container{
					{
						Name:  "secret-volume-test",
						Image: "gcr.io/google_containers/mounttest:0.7",
						Args: []string{
							"--file_content=/etc/secret-volume/data-1",
							"--file_mode=/etc/secret-volume/data-1"},
						VolumeMounts: []api.VolumeMount{
							{
								Name:      volumeName,
								MountPath: volumeMountPath,
							},
						},
					},
				},
				RestartPolicy: api.RestartPolicyNever,
			},
		}

		f.TestContainerOutput("consume secrets", pod, 0, []string{
			"content of file \"/etc/secret-volume/data-1\": value-1",
			"mode of file \"/etc/secret-volume/data-1\": -r--------",
		})
	})

	It("should be consumable in multiple volumes in a pod [Conformance]", func() {
		// This test ensures that the same secret can be mounted in multiple
		// volumes in the same pod.  This test case exists to prevent
		// regressions that break this use-case.
		var (
			name             = "secret-test-" + string(uuid.NewUUID())
			volumeName       = "secret-volume"
			volumeMountPath  = "/etc/secret-volume"
			volumeName2      = "secret-volume-2"
			volumeMountPath2 = "/etc/secret-volume-2"
			secret           = secretForTest(f.Namespace.Name, name)
		)

		By(fmt.Sprintf("Creating secret with name %s", secret.Name))
		var err error
		if secret, err = f.Client.Secrets(f.Namespace.Name).Create(secret); err != nil {
			framework.Failf("unable to create test secret %s: %v", secret.Name, err)
		}

		pod := &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: "pod-secrets-" + string(uuid.NewUUID()),
			},
			Spec: api.PodSpec{
				Volumes: []api.Volume{
					{
						Name: volumeName,
						VolumeSource: api.VolumeSource{
							Secret: &api.SecretVolumeSource{
								SecretName: name,
							},
						},
					},
					{
						Name: volumeName2,
						VolumeSource: api.VolumeSource{
							Secret: &api.SecretVolumeSource{
								SecretName: name,
							},
						},
					},
				},
				Containers: []api.Container{
					{
						Name:  "secret-volume-test",
						Image: "gcr.io/google_containers/mounttest:0.7",
						Args: []string{
							"--file_content=/etc/secret-volume/data-1",
							"--file_mode=/etc/secret-volume/data-1"},
						VolumeMounts: []api.VolumeMount{
							{
								Name:      volumeName,
								MountPath: volumeMountPath,
								ReadOnly:  true,
							},
							{
								Name:      volumeName2,
								MountPath: volumeMountPath2,
								ReadOnly:  true,
							},
						},
					},
				},
				RestartPolicy: api.RestartPolicyNever,
			},
		}

		f.TestContainerOutput("consume secrets", pod, 0, []string{
			"content of file \"/etc/secret-volume/data-1\": value-1",
			"mode of file \"/etc/secret-volume/data-1\": -rw-r--r--",
		})
	})

	It("should be consumable from pods in env vars [Conformance]", func() {
		name := "secret-test-" + string(uuid.NewUUID())
		secret := secretForTest(f.Namespace.Name, name)

		By(fmt.Sprintf("Creating secret with name %s", secret.Name))
		var err error
		if secret, err = f.Client.Secrets(f.Namespace.Name).Create(secret); err != nil {
			framework.Failf("unable to create test secret %s: %v", secret.Name, err)
		}

		pod := &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: "pod-secrets-" + string(uuid.NewUUID()),
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:    "secret-env-test",
						Image:   "gcr.io/google_containers/busybox:1.24",
						Command: []string{"sh", "-c", "env"},
						Env: []api.EnvVar{
							{
								Name: "SECRET_DATA",
								ValueFrom: &api.EnvVarSource{
									SecretKeyRef: &api.SecretKeySelector{
										LocalObjectReference: api.LocalObjectReference{
											Name: name,
										},
										Key: "data-1",
									},
								},
							},
						},
					},
				},
				RestartPolicy: api.RestartPolicyNever,
			},
		}

		f.TestContainerOutput("consume secrets", pod, 0, []string{
			"SECRET_DATA=value-1",
		})
	})
})

func secretForTest(namespace, name string) *api.Secret {
	return &api.Secret{
		ObjectMeta: api.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			"data-1": []byte("value-1\n"),
			"data-2": []byte("value-2\n"),
			"data-3": []byte("value-3\n"),
		},
	}
}

func doSecretE2E(f *framework.Framework, defaultMode *int32) {
	var (
		name            = "secret-test-" + string(uuid.NewUUID())
		volumeName      = "secret-volume"
		volumeMountPath = "/etc/secret-volume"
		secret          = secretForTest(f.Namespace.Name, name)
	)

	By(fmt.Sprintf("Creating secret with name %s", secret.Name))
	defer func() {
		By("Cleaning up the secret")
		if err := f.Client.Secrets(f.Namespace.Name).Delete(secret.Name); err != nil {
			framework.Failf("unable to delete secret %v: %v", secret.Name, err)
		}
	}()
	var err error
	if secret, err = f.Client.Secrets(f.Namespace.Name).Create(secret); err != nil {
		framework.Failf("unable to create test secret %s: %v", secret.Name, err)
	}

	pod := &api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name: "pod-secrets-" + string(uuid.NewUUID()),
		},
		Spec: api.PodSpec{
			Volumes: []api.Volume{
				{
					Name: volumeName,
					VolumeSource: api.VolumeSource{
						Secret: &api.SecretVolumeSource{
							SecretName: name,
						},
					},
				},
			},
			Containers: []api.Container{
				{
					Name:  "secret-volume-test",
					Image: "gcr.io/google_containers/mounttest:0.7",
					Args: []string{
						"--file_content=/etc/secret-volume/data-1",
						"--file_mode=/etc/secret-volume/data-1"},
					VolumeMounts: []api.VolumeMount{
						{
							Name:      volumeName,
							MountPath: volumeMountPath,
						},
					},
				},
			},
			RestartPolicy: api.RestartPolicyNever,
		},
	}

	if defaultMode != nil {
		pod.Spec.Volumes[0].VolumeSource.Secret.DefaultMode = defaultMode
	} else {
		mode := int32(0644)
		defaultMode = &mode
	}

	modeString := fmt.Sprintf("%v", os.FileMode(*defaultMode))
	expectedOutput := []string{
		"content of file \"/etc/secret-volume/data-1\": value-1",
		"mode of file \"/etc/secret-volume/data-1\": " + modeString,
	}

	f.TestContainerOutput("consume secrets", pod, 0, expectedOutput)
}
