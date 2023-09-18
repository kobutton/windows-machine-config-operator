package vsphere

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/pkg/errors"

	config "github.com/openshift/api/config/v1"
	mapi "github.com/openshift/api/machine/v1beta1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/windows-machine-config-operator/test/e2e/clusterinfo"
	"github.com/openshift/windows-machine-config-operator/test/e2e/providers/machineset"
)

const defaultCredentialsSecretName = "vsphere-cloud-credentials"

// Provider is a provider struct for testing vSphere
type Provider struct {
	oc *clusterinfo.OpenShift
	*config.InfrastructureStatus
}

// New returns a new vSphere provider struct with the given client set and ssh key pair
func New(clientset *clusterinfo.OpenShift, infraStatus *config.InfrastructureStatus) (*Provider, error) {
	return &Provider{
		oc:                   clientset,
		InfrastructureStatus: infraStatus,
	}, nil
}

// newVSphereMachineProviderSpec returns a vSphereMachineProviderSpec generated from the inputs, or an error
func (p *Provider) newVSphereMachineProviderSpec() (*mapi.VSphereMachineProviderSpec, error) {
	existingProviderSpec, err := p.getProviderSpecFromExistingMachineSet()
	if err != nil {
		return nil, err
	}
	log.Printf("creating machineset provider spec which targets %s with network %s\n",
		existingProviderSpec.Workspace.Server, existingProviderSpec.Network)

	// The template is an image which has been properly sysprepped.  The image is derived from an environment variable
	// defined in the job spec.
	vmTemplate := os.Getenv("VM_TEMPLATE")
	if vmTemplate == "" {
		vmTemplate = "windows-golden-images/windows-server-2022-template-ipv6-disabled"
	}

	log.Printf("creating machineset based on template %s\n", vmTemplate)

	return &mapi.VSphereMachineProviderSpec{
		TypeMeta: meta.TypeMeta{
			APIVersion: "vsphereprovider.openshift.io/v1beta1",
			Kind:       "VSphereMachineProviderSpec",
		},
		CredentialsSecret: &core.LocalObjectReference{
			Name: defaultCredentialsSecretName,
		},
		DiskGiB:           int32(128),
		MemoryMiB:         int64(16384),
		Network:           existingProviderSpec.Network,
		NumCPUs:           int32(4),
		NumCoresPerSocket: int32(1),
		Template:          vmTemplate,
		Workspace:         existingProviderSpec.Workspace,
	}, nil
}

// getProviderSpecFromExistingMachineSet returns the providerSpec of an existing machineset provisioned during installation
func (p *Provider) getProviderSpecFromExistingMachineSet() (*mapi.VSphereMachineProviderSpec, error) {
	listOptions := meta.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-cluster=" +
		p.InfrastructureName}
	machineSets, err := p.oc.Machine.MachineSets("openshift-machine-api").List(context.TODO(), listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get machinesets")
	}

	if len(machineSets.Items) == 0 {
		return nil, errors.Wrap(err, "no matching machinesets found")
	}

	machineSet := machineSets.Items[0]
	providerSpecRaw := machineSet.Spec.Template.Spec.ProviderSpec.Value
	if providerSpecRaw == nil || providerSpecRaw.Raw == nil {
		return nil, errors.Wrap(err, "no provider spec found")
	}
	var providerSpec mapi.VSphereMachineProviderSpec
	err = json.Unmarshal(providerSpecRaw.Raw, &providerSpec)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal providerSpec")
	}

	return &providerSpec, nil
}

// GenerateMachineSet generates the MachineSet object which is vSphere provider specific
func (p *Provider) GenerateMachineSet(withWindowsLabel bool, replicas int32) (*mapi.MachineSet, error) {
	// create new machine provider spec for deploying Windows node
	providerSpec, err := p.newVSphereMachineProviderSpec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new vSphere machine provider spec")
	}

	rawProviderSpec, err := json.Marshal(providerSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal vSphere machine provider spec")
	}

	return machineset.New(rawProviderSpec, p.InfrastructureName, replicas, withWindowsLabel), nil
}

func (p *Provider) GetType() config.PlatformType {
	return config.VSpherePlatformType
}
