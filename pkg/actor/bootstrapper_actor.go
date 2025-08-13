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

package actor

import (
	"context"
	"fmt"

	ravendbv1alpha1 "ravendb-operator/api/v1alpha1"
	"ravendb-operator/pkg/resource"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BootstrapperActor struct {
	builder resource.PerClusterBuilder
}

func NewBootstrapperActor(builder resource.PerClusterBuilder) PerClusterActor {
	return &BootstrapperActor{builder: builder}
}

func (a *BootstrapperActor) Name() string {
	return "BootstrapperActor"
}

func (a *BootstrapperActor) Act(ctx context.Context, cluster *ravendbv1alpha1.RavenDBCluster, c client.Client, scheme *runtime.Scheme) error {
	bs, err := a.builder.Build(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to build Bootstrapper job: %w", err)
	}

	if err := applyResource(ctx, c, scheme, bs); err != nil {
		return fmt.Errorf("failed to apply Bootstrapper job: %w", err)
	}
	return nil
}

func (a *BootstrapperActor) ShouldAct(cluster *ravendbv1alpha1.RavenDBCluster) bool {
	AutomaticClusterSetup := cluster.Spec.AutomaticClusterSetupSpec

	return AutomaticClusterSetup != nil
}
