package docker

import (
	"fmt"
	"strings"
)

type DockerClusterToCommunityConverter struct {
}

func (c *DockerClusterToCommunityConverter) ClusterToCommunityIP(clusterIP string) (communityIP string, err error) {
	//clusterIP has the following format: cluster0, cluster1, etc.
	// we need to conver that to community0, community1, etc.

	if !strings.HasPrefix(clusterIP, "cluster") {
		return "", fmt.Errorf("clusterIP %s does not have the correct format", clusterIP)
	}

	communityIP = "community" + clusterIP[len("cluster"):]

	return communityIP, nil
}
