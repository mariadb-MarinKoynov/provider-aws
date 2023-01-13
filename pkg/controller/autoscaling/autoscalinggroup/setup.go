/*
Copyright 2021 The Crossplane Authors.

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

package autoscalinggroup

import (
	"context"

	svcsdk "github.com/aws/aws-sdk-go/service/autoscaling"
	svcapitypes "github.com/crossplane-contrib/provider-aws/apis/autoscaling/v1alpha1"
	"github.com/crossplane-contrib/provider-aws/apis/v1alpha1"
	awsclient "github.com/crossplane-contrib/provider-aws/pkg/clients"
	"github.com/crossplane-contrib/provider-aws/pkg/features"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupAutoscalingGroup adds a controller that reconciles Auto Scaling Group.
func SetupAutoscalingGroup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(svcapitypes.AutoScalingGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), v1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&svcapitypes.AutoScalingGroup{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(svcapitypes.AutoScalingGroupGroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube: mgr.GetClient(),
				opts: []option{
					func(e *external) {
						e.preCreate = preCreate
						e.preUpdate = preUpdate
						e.preObserve = preObserve
						e.postObserve = postObserve
						e.preDelete = preDelete
					},
				},
			}),
			managed.WithPollInterval(o.PollInterval),
			managed.WithLogger(o.Logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...)))
}

func preCreate(_ context.Context, cr *svcapitypes.AutoScalingGroup, casgi *svcsdk.CreateAutoScalingGroupInput) error {
	casgi.AutoScalingGroupName = awsclient.String(meta.GetExternalName(cr))
	return nil
}

func preUpdate(_ context.Context, cr *svcapitypes.AutoScalingGroup, uasgi *svcsdk.UpdateAutoScalingGroupInput) error {
	uasgi.AutoScalingGroupName = awsclient.String(meta.GetExternalName(cr))
	return nil
}

func preObserve(_ context.Context, cr *svcapitypes.AutoScalingGroup, oasgi *svcsdk.DescribeAutoScalingGroupsInput) error {
	oasgi.AutoScalingGroupNames = []*string{awsclient.String(meta.GetExternalName(cr))}
	return nil
}

func postObserve(_ context.Context, cr *svcapitypes.AutoScalingGroup, _ *svcsdk.DescribeAutoScalingGroupsOutput,
	eo managed.ExternalObservation, err error) (managed.ExternalObservation, error) {
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.SetConditions(xpv1.Available())
	return eo, nil
}

func preDelete(_ context.Context, cr *svcapitypes.AutoScalingGroup, dasgi *svcsdk.DeleteAutoScalingGroupInput) (bool, error) {
	dasgi.AutoScalingGroupName = awsclient.String(meta.GetExternalName(cr))
	return false, nil
}
