// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/cloudbase/garm-provider-common/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestToProviderInstance(t *testing.T) {
	inst := &computepb.Instance{
		Name:   proto.String("garm-abc"),
		Zone:   proto.String("https://www.googleapis.com/compute/v1/projects/p/zones/us-central1-f"),
		Status: proto.String("TERMINATED"),
		Labels: map[string]string{"ostype": "linux"},
		Disks:  []*computepb.AttachedDisk{{Architecture: proto.String("ARM64")}},
		Metadata: &computepb.Metadata{Items: []*computepb.Items{
			{Key: proto.String("runner_name"), Value: proto.String("Garm-ABC")},
		}},
	}
	pi, err := ToProviderInstance(inst)
	require.NoError(t, err)
	assert.Equal(t, "us-central1-f/garm-abc", pi.ProviderID)
	assert.Equal(t, "Garm-ABC", pi.Name)
	assert.Equal(t, params.Linux, pi.OSType)
	assert.Equal(t, params.Arm64, pi.OSArch)
	assert.Equal(t, params.InstanceStopped, pi.Status, "a preempted Spot VM is reported stopped")

	_, err = ToProviderInstance(&computepb.Instance{Name: proto.String("x")})
	assert.Error(t, err)
}

func TestSplitProviderID(t *testing.T) {
	zone, name, ok := SplitProviderID("us-central1-b/Garm-X")
	assert.True(t, ok)
	assert.Equal(t, "us-central1-b", zone)
	assert.Equal(t, "garm-x", name)
	for _, bad := range []string{"garm-x", "/garm-x", "z/", "z/a/b"} {
		_, _, ok := SplitProviderID(bad)
		assert.False(t, ok, bad)
	}
}
