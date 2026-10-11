// SPDX-License-Identifier: Apache-2.0

package mig

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/ut0mt8/garm-provider-gcp-mig/config"
	"golang.org/x/oauth2/google"
	gcompute "google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
)

// API is the subset of Compute Engine calls the MIG manager needs. It hides
// iterators and long-running operations so the manager logic can be tested
// with a simple fake.
type API interface {
	CreateInstance(ctx context.Context, project, region, mig, name string, metadata map[string]string) error
	ListManaged(ctx context.Context, project, region, mig string) ([]*computepb.ManagedInstance, error)
	DeleteManaged(ctx context.Context, project, region, mig string, instanceURLs []string) error
	DeletePerInstanceConfigs(ctx context.Context, project, region, mig string, names []string) error
	GetInstance(ctx context.Context, project, zone, name string) (*computepb.Instance, error)
	SetLabels(ctx context.Context, project, zone, name, fingerprint string, labels map[string]string) error
	ListInstances(ctx context.Context, project, filter string) ([]*computepb.Instance, error)
	StopInstance(ctx context.Context, project, zone, name string) error
	StartInstance(ctx context.Context, project, zone, name string) error
	GetMIG(ctx context.Context, project, region, mig string) (*computepb.InstanceGroupManager, error)
}

// ErrNotFound is returned when an instance (or per-instance config) does not exist.
var ErrNotFound = errors.New("not found")

func isNotFound(err error) bool {
	var apiErr *apierror.APIError
	return errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusNotFound
}

type gcpAPI struct {
	instances *compute.InstancesClient
	igms      *compute.RegionInstanceGroupManagersClient
}

// NewAPI builds the real Compute Engine API, with the same credential rules
// as the upstream client (credentials_file if set, else default credentials).
func NewAPI(ctx context.Context, cfg *config.Config) (API, error) {
	var opts []option.ClientOption
	if cfg.CredentialsFile != "" {
		jsonKey, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read JSON key file: %w", err)
		}
		jwtCfg, err := google.JWTConfigFromJSON(jsonKey, gcompute.CloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("failed to create JWT config: %w", err)
		}
		opts = append(opts, option.WithHTTPClient(jwtCfg.Client(ctx)))
	} else {
		creds, err := google.FindDefaultCredentials(ctx, gcompute.CloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	instances, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating instances client: %w", err)
	}
	igms, err := compute.NewRegionInstanceGroupManagersRESTClient(ctx, opts...)
	if err != nil {
		_ = instances.Close()
		return nil, fmt.Errorf("error creating region instance group managers client: %w", err)
	}
	return &gcpAPI{instances: instances, igms: igms}, nil
}

func (a *gcpAPI) CreateInstance(ctx context.Context, project, region, mig, name string, metadata map[string]string) error {
	op, err := a.igms.CreateInstances(ctx, &computepb.CreateInstancesRegionInstanceGroupManagerRequest{
		Project:              project,
		Region:               region,
		InstanceGroupManager: mig,
		RegionInstanceGroupManagersCreateInstancesRequestResource: &computepb.RegionInstanceGroupManagersCreateInstancesRequest{
			Instances: []*computepb.PerInstanceConfig{{
				Name:           proto.String(name),
				PreservedState: &computepb.PreservedState{Metadata: metadata},
			}},
		},
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (a *gcpAPI) ListManaged(ctx context.Context, project, region, mig string) ([]*computepb.ManagedInstance, error) {
	it := a.igms.ListManagedInstances(ctx, &computepb.ListManagedInstancesRegionInstanceGroupManagersRequest{
		Project:              project,
		Region:               region,
		InstanceGroupManager: mig,
	})
	var out []*computepb.ManagedInstance
	for {
		mi, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, mi)
	}
}

func (a *gcpAPI) DeleteManaged(ctx context.Context, project, region, mig string, instanceURLs []string) error {
	op, err := a.igms.DeleteInstances(ctx, &computepb.DeleteInstancesRegionInstanceGroupManagerRequest{
		Project:              project,
		Region:               region,
		InstanceGroupManager: mig,
		RegionInstanceGroupManagersDeleteInstancesRequestResource: &computepb.RegionInstanceGroupManagersDeleteInstancesRequest{
			Instances:                      instanceURLs,
			SkipInstancesOnValidationError: proto.Bool(true),
		},
	})
	if err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return err
	}
	return op.Wait(ctx)
}

func (a *gcpAPI) DeletePerInstanceConfigs(ctx context.Context, project, region, mig string, names []string) error {
	op, err := a.igms.DeletePerInstanceConfigs(ctx, &computepb.DeletePerInstanceConfigsRegionInstanceGroupManagerRequest{
		Project:              project,
		Region:               region,
		InstanceGroupManager: mig,
		RegionInstanceGroupManagerDeleteInstanceConfigReqResource: &computepb.RegionInstanceGroupManagerDeleteInstanceConfigReq{
			Names: names,
		},
	})
	if err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return err
	}
	return op.Wait(ctx)
}

func (a *gcpAPI) GetInstance(ctx context.Context, project, zone, name string) (*computepb.Instance, error) {
	inst, err := a.instances.Get(ctx, &computepb.GetInstanceRequest{Project: project, Zone: zone, Instance: name})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return inst, nil
}

func (a *gcpAPI) SetLabels(ctx context.Context, project, zone, name, fingerprint string, labels map[string]string) error {
	op, err := a.instances.SetLabels(ctx, &computepb.SetLabelsInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
		InstancesSetLabelsRequestResource: &computepb.InstancesSetLabelsRequest{
			LabelFingerprint: proto.String(fingerprint),
			Labels:           labels,
		},
	})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (a *gcpAPI) ListInstances(ctx context.Context, project, filter string) ([]*computepb.Instance, error) {
	it := a.instances.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{
		Project:              project,
		Filter:               proto.String(filter),
		ReturnPartialSuccess: proto.Bool(true),
	})
	var out []*computepb.Instance
	for {
		pair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if pair.Value != nil {
			out = append(out, pair.Value.Instances...)
		}
	}
}

func (a *gcpAPI) StopInstance(ctx context.Context, project, zone, name string) error {
	op, err := a.instances.Stop(ctx, &computepb.StopInstanceRequest{Project: project, Zone: zone, Instance: name})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (a *gcpAPI) StartInstance(ctx context.Context, project, zone, name string) error {
	op, err := a.instances.Start(ctx, &computepb.StartInstanceRequest{Project: project, Zone: zone, Instance: name})
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (a *gcpAPI) GetMIG(ctx context.Context, project, region, mig string) (*computepb.InstanceGroupManager, error) {
	igm, err := a.igms.Get(ctx, &computepb.GetRegionInstanceGroupManagerRequest{
		Project:              project,
		Region:               region,
		InstanceGroupManager: mig,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return igm, nil
}
