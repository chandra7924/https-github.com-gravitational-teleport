/*
Copyright 2020 Gravitational, Inc.

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

package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRole_GetKubeResources(t *testing.T) {
	kubeLabels := Labels{
		Wildcard: {Wildcard},
	}
	type args struct {
		version   string
		labels    Labels
		resources []KubernetesResource
	}
	tests := []struct {
		name                string
		args                args
		want                []KubernetesResource
		assertErrorCreation require.ErrorAssertionFunc
	}{
		// TODO(tigrato): add more tests once we support other kubernetes resources.
		{
			name: "v7 with error",
			args: args{
				version: V7,
				labels:  kubeLabels,
				resources: []KubernetesResource{
					{
						Kind:      "invalid resource",
						Namespace: "test",
						Name:      "test",
					},
				},
			},
			assertErrorCreation: require.Error,
		},
		{
			name: "v7",
			args: args{
				version: V7,
				labels:  kubeLabels,
				resources: []KubernetesResource{
					{
						Kind:      KindKubePod,
						Namespace: "test",
						Name:      "test",
					},
				},
			},
			assertErrorCreation: require.NoError,
			want: []KubernetesResource{
				{
					Kind:      KindKubePod,
					Namespace: "test",
					Name:      "test",
				},
			},
		},
		{
			name: "v6 to v7 with wildcard",
			args: args{
				version: V6,
				labels:  kubeLabels,
				resources: []KubernetesResource{
					{
						Kind:      KindKubePod,
						Namespace: Wildcard,
						Name:      Wildcard,
					},
				},
			},
			assertErrorCreation: require.NoError,
			want: []KubernetesResource{
				{
					Kind:      Wildcard,
					Namespace: Wildcard,
					Name:      Wildcard,
				},
			},
		},
		{
			name: "v6 to v7 without wildcard",
			args: args{
				version: V6,
				labels:  kubeLabels,
				resources: []KubernetesResource{
					{
						Kind:      KindKubePod,
						Namespace: "test",
						Name:      "test",
					},
				},
			},
			assertErrorCreation: require.NoError,
			want: []KubernetesResource{
				{
					Kind:      KindKubePod,
					Namespace: "test",
					Name:      "test",
				},
			},
		},
		{
			name: "v5 to v7: populate with defaults.",
			args: args{
				version:   V5,
				labels:    kubeLabels,
				resources: nil,
			},
			assertErrorCreation: require.NoError,
			want: []KubernetesResource{
				{
					Kind:      Wildcard,
					Namespace: Wildcard,
					Name:      Wildcard,
				},
			},
		},
		{
			name: "v5 to v7 without kube labels",
			args: args{
				version:   V5,
				resources: nil,
			},
			assertErrorCreation: require.NoError,
			want:                nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRoleWithVersion(
				"test",
				tt.args.version,
				RoleSpecV6{
					Allow: RoleConditions{
						Namespaces:          []string{"default"},
						KubernetesLabels:    tt.args.labels,
						KubernetesResources: tt.args.resources,
					},
				},
			)
			tt.assertErrorCreation(t, err)
			if err != nil {
				return
			}
			got := r.GetKubeResources(Allow)
			require.Equal(t, tt.want, got)
			got = r.GetKubeResources(Deny)
			require.Empty(t, got)
		})
	}
}
