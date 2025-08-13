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

package validator

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bootstrapperValidator struct {
	client client.Reader
}

func NewBootstrapperValidator(c client.Reader) *bootstrapperValidator {
	return &bootstrapperValidator{client: c}
}

func (v *bootstrapperValidator) Name() string {
	return "bootstrapper-validator"
}

func (v *bootstrapperValidator) ValidateCreate(ctx context.Context, c ClusterAdapter) error {
	var errs []string

	if !c.IsBootstrapperSet() {
		return nil
	}

	if c.GetMode() == "LetsEncrypt" && c.GetCACertSecretRef() != nil {
		errs = append(errs, "caCertSecretRef must not be set when mode is LetsEncrypt")
	}

	if c.GetMode() == "None" && c.GetCACertSecretRef() == nil {
		errs = append(errs, "caCertSecretRef must be provided when mode is None and clusterBootstrapper is set")
	}

	tags := c.GetNodeTags()
	leader := c.GetBootsrapperLeader()
	watchers := c.GetBootsrapperWatchers()

	tagSet := make(map[string]struct{})
	for _, nodeTag := range tags {
		tagSet[nodeTag] = struct{}{}
	}

	if _, exists := tagSet[leader]; !exists {
		errs = append(errs, fmt.Sprintf("leader tag %q not found in spec.nodes", leader))
	}

	for _, w := range watchers {
		if _, exists := tagSet[w]; !exists {
			errs = append(errs, fmt.Sprintf("watcher tag %q not found in spec.nodes", w))
		}
		if w == leader {
			errs = append(errs, fmt.Sprintf("watcher tag %q cannot also be the leader", w))
		}
	}

	watcherSeen := make(map[string]struct{})
	for _, w := range watchers {
		if _, seen := watcherSeen[w]; seen {
			errs = append(errs, fmt.Sprintf("duplicate watcher tag %q", w))
		}
		watcherSeen[w] = struct{}{}
	}

	if len(watchers)+1 > len(tags) {
		errs = append(errs, "leader + watchers count exceeds number of nodes in spec.nodes")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}

	return nil

}

func (v *bootstrapperValidator) ValidateUpdate(ctx context.Context, _, newC ClusterAdapter) error {
	return v.ValidateCreate(ctx, newC)
}
