/*
Copyright 2023 Gravitational, Inc.

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
	"encoding/json"

	"github.com/gravitational/trace"
)

// UnmarshalYAML converts bytes into an AWSSSM struct.
// This is required because we can't define yaml tags in proto files (AWSSSM is defined in types.pb.go).
// `sigs.k8s.io/yaml` can't be used because it looks only for json tags and ignores the yaml ones
// To be able to use `sigs.k8s.io/yaml`, we would need to add json tags to all the [lib/config.FileConfig] fields
func (s *AWSSSM) UnmarshalYAML(unmarshal func(interface{}) error) error {
	val := &struct {
		DocumentName string `yaml:"document_name"`
	}{}
	err := unmarshal(&val)
	if err != nil {
		return trace.Wrap(err)
	}

	s.DocumentName = val.DocumentName
	return nil
}

// UnmarshalYAML converts bytes into an AzureJoinParams struct.
// This is required because we can't define yaml tags in proto files (AzureJoinParams is defined in types.pb.go).
// `sigs.k8s.io/yaml` can't be used because it looks only for json tags and ignores the yaml ones
// To be able to use `sigs.k8s.io/yaml`, we would need to add json tags to all the [lib/config.FileConfig] fields
func (s *AzureJoinParams) UnmarshalYAML(unmarshal func(interface{}) error) error {
	val := &struct {
		ClientID string `yaml:"client_id"`
	}{}
	err := unmarshal(&val)
	if err != nil {
		return trace.Wrap(err)
	}

	s.ClientID = val.ClientID
	return nil
}

// UnmarshalYAML converts bytes into an GCPMatcher struct.
// This is required because we can't define yaml tags in proto files (GCPMatcher is defined in types.pb.go).
// `sigs.k8s.io/yaml` can't be used because it looks only for json tags and ignores the yaml ones
// To be able to use `sigs.k8s.io/yaml`, we would need to add json tags to all the [lib/config.FileConfig] fields
func (s *GCPMatcher) UnmarshalYAML(unmarshal func(interface{}) error) error {
	val := &struct {
		// Types are GKE resource types to match: "gke".
		Types []string `yaml:"types,omitempty"`
		// Locations are GCP locations to search resources for.
		Locations []string `yaml:"locations,omitempty"`
		// Tags are GCP labels to match.
		Tags Labels `yaml:"tags,omitempty"`
		// ProjectIDs are the GCP project IDs where the resources are deployed.
		ProjectIDs []string `yaml:"project_ids,omitempty"`
	}{}
	err := unmarshal(&val)
	if err != nil {
		return trace.Wrap(err)
	}

	s.Types = val.Types
	s.Locations = val.Locations
	s.Tags = val.Tags.Clone()
	s.ProjectIDs = val.ProjectIDs

	return nil
}

// MarshalYAML converts GCPMatcher into YAML.
func (s *GCPMatcher) MarshalYAML() (interface{}, error) {
	return json.Marshal(s)
}
