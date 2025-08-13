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

package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func baseClusterForAutoBootstrapTypesTest(name string) *RavenDBCluster {
	email := "user@example.com"
	certSecretRef := "ravendb-certs-a"
	return &RavenDBCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: RavenDBClusterSpec{
			Image:            "ravendb/ravendb:latest",
			ImagePullPolicy:  "Always",
			Mode:             "None",
			Email:            &email,
			LicenseSecretRef: "license-secret",
			Domain:           "example.com",
			Nodes: []RavenDBNode{
				{
					Tag:                "A",
					PublicServerUrl:    "https://a.example.com",
					PublicServerUrlTcp: "tcp://a-tcp.example.com",
					CertSecretRef:      &certSecretRef,
				},
				{
					Tag:                "B",
					PublicServerUrl:    "https://b.example.com",
					PublicServerUrlTcp: "tcp://b-tcp.example.com",
				},
				{
					Tag:                "C",
					PublicServerUrl:    "https://c.example.com",
					PublicServerUrlTcp: "tcp://c-tcp.example.com",
				},
			},
			StorageSpec: StorageSpec{
				Data: VolumeSpec{Size: "5Gi"},
			},
			AutomaticClusterSetupSpec: &AutomaticClusterSetupSpec{
				Leader:   "A",
				Watchers: &[]string{"B", "C"},
			},
		},
	}
}

func TestAutomaticClusterSetupSchema(t *testing.T) {
	testCases := []SpecValidationCase{
		{
			Name:        "valid leader + watchers",
			Modify:      func(spec *RavenDBClusterSpec) {},
			ExpectError: false,
		},
		{
			Name: "missing leader (empty string)",
			Modify: func(spec *RavenDBClusterSpec) {
				spec.AutomaticClusterSetupSpec.Leader = ""
			},
			ExpectError: true,
			ErrorParts:  []string{"spec.automaticClusterSetup.leader"},
		},
		{
			Name: "watchers omitted (nil pointer) is allowed",
			Modify: func(spec *RavenDBClusterSpec) {
				spec.AutomaticClusterSetupSpec.Watchers = nil
			},
			ExpectError: false,
		},
		{
			Name: "watchers present but empty (minItems=1)",
			Modify: func(spec *RavenDBClusterSpec) {
				empty := []string{}
				spec.AutomaticClusterSetupSpec.Watchers = &empty
			},
			ExpectError: true,
			ErrorParts:  []string{"spec.automaticClusterSetup.watchers"},
		},
		{
			Name: "automaticClusterSetup omitted entirely",
			Modify: func(spec *RavenDBClusterSpec) {
				spec.AutomaticClusterSetupSpec = nil
			},
			ExpectError: false,
		},
	}

	runSpecValidationTest(t, baseClusterForAutoBootstrapTypesTest, testCases)
}
