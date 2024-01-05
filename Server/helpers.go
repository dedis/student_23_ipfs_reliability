package Server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (s *Server) getAllPeers() (map[string]string, map[string]CommunityNode, []string, error) {
	url := fmt.Sprintf("http://%s/peers", s.discoveryAddress)
	resp, err := http.Get(url)

	if err != nil {
		return nil, nil, nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, nil, err
	}

	var nodes CommunitiesMap
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, nil, nil, err
	}

	// remove self from the list
	delete(nodes, s.address)

	clusterToCommunity := make(map[string]string)
	// Get the list of peer names
	peerNames := make([]string, 0, len(nodes))
	for name, info := range nodes {
		peerNames = append(peerNames, name)
		clusterToCommunity[info.ClusterIP] = name
	}

	return clusterToCommunity, nodes, peerNames, nil
}

// convert cluster ip to community address
func (s *Server) getCommunityAddress(clusterIP string) (string, error) {
	baseURL := fmt.Sprintf("http://%s/cluster-to-community", s.discoveryAddress)

	// Build the query parameters
	params := url.Values{}
	params.Add("clusterIP", clusterIP)

	// Construct the final URL with query parameters
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Send the GET request
	resp, err := http.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	return string(body), nil
}

func (s *Server) printState() {
	s.stateMux.Lock()
	defer s.stateMux.Unlock()

	fmt.Printf("State: %+v\n", s.state.String())
}

func (state *State) String() string {
	return "impl state printing"
}
