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

/*
when user change the size of the pvc, k8s will automatically resize the pvc (assuming the storageClass support resizing which is most of them).
However, even though Kubernetes resizes individual PVCs when the spec change, which results in the pvc being resized,
it does not automatically update the status of the pvc which means differnece between the actual state and the desired state.

will be addressed on RavenDB-24331 Kubernetes Operator: Bind storage solutions

TODO: //
*/
