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

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func applyResource(ctx context.Context, c client.Client, scheme *runtime.Scheme, obj client.Object) error {
	log := ctrl.LoggerFrom(ctx)
	key := client.ObjectKeyFromObject(obj)
	existing := obj.DeepCopyObject().(client.Object)

	err := c.Get(ctx, key, existing)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get resource: %w", err)
		}

		if err := c.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}
		log.Info("Created resource", "name", key.Name)
		return nil
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	if err := c.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}
	log.Info("Updated resource", "name", key.Name)
	return nil
}
