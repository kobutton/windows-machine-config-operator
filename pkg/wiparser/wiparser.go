package wiparser

import (
	"net"
	"strings"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"

	"github.com/openshift/windows-machine-config-operator/pkg/instances"
)

// InstanceConfigMap is the name of the ConfigMap where VMs to be configured should be described.
const InstanceConfigMap = "windows-instances"

// Parse returns the list of instances specified in the Windows instances data. This function should be passed a list
// of Nodes in the cluster, as each instance returned will contain a reference to its associated Node, if it has one
// in the given NodeList. If an instance does not have an associated node from the NodeList, the node reference will
// be nil.
func Parse(instancesData map[string]string, nodes *core.NodeList) ([]*instances.InstanceInfo, error) {
	if nodes == nil {
		return nil, errors.New("nodes cannot be nil")
	}
	instanceList := make([]*instances.InstanceInfo, 0)
	// Get information about the instances from each entry. The expected key/value format for each entry is:
	// <address>: username=<username>
	for address, data := range instancesData {
		if err := validateAddress(address); err != nil {
			return nil, errors.Wrapf(err, "invalid address %s", address)
		}
		username, err := extractUsername(data)
		if err != nil {
			return instanceList, errors.Wrapf(err, "unable to get username for %s", address)
		}

		// Get the associated node if the described instance has one
		node, _ := findNode(address, nodes)
		instanceList = append(instanceList, instances.NewInstanceInfo(address, username, "", node))
	}
	return instanceList, nil
}

// validateAddress checks that the given address is either an ipv4 address, or resolves to any ip address
func validateAddress(address string) error {
	// first check if address is an IP address
	if parsedAddr := net.ParseIP(address); parsedAddr != nil {
		if parsedAddr.To4() != nil {
			return nil
		}
		// if the address parses into an IP but is not ipv4 it must be ipv6
		return errors.Errorf("ipv6 is not supported")
	}
	// Do a check that the DNS provided is valid
	addressList, err := net.LookupHost(address)
	if err != nil {
		return errors.Wrapf(err, "error looking up DNS")
	}
	if len(addressList) == 0 {
		return errors.Errorf("DNS did not resolve to an address")
	}
	return nil
}

// findNode returns a pointer to the node with an address matching the given address and a bool indicating if the node
// was found or not.
func findNode(address string, nodes *core.NodeList) (*core.Node, bool) {
	for _, node := range nodes.Items {
		for _, nodeAddress := range node.Status.Addresses {
			if address == nodeAddress.Address {
				return &node, true
			}
		}
	}
	return nil, false
}

// GetNodeUsername retrieves the username associated with the given node from the instance ConfigMap data
func GetNodeUsername(instancesData map[string]string, node *core.Node) (string, error) {
	if node == nil {
		return "", errors.New("cannot get username for nil node")
	}
	// Find entry in ConfigMap that is associated to node via address
	for _, address := range node.Status.Addresses {
		if value, found := instancesData[address.Address]; found {
			return extractUsername(value)
		}
	}
	return "", errors.Errorf("unable to find instance associated with node %s", node.GetName())
}

// extractUsername returns the username string from data in the form username=<username>
func extractUsername(value string) (string, error) {
	splitData := strings.SplitN(value, "=", 2)
	if len(splitData) == 0 || splitData[0] != "username" {
		return "", errors.New("data has an incorrect format")
	}
	return splitData[1], nil
}
