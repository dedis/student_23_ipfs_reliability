package client

import (
	"encoding/json"
	ipfscluster "ipfs-alpha-entanglement-code/ipfs-cluster"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"

	"golang.org/x/xerrors"
)

type Metadata struct {
	Alpha int
	S     int
	P     int

	OriginalFileCID string
	TreeCIDs        []string
	NumBlocks       int // N
	MaxChildren     int // K
	Leaves          int // L
	Depth           int // D

	MaxParityChildren int // K Parity

	RootCID string

	DataCIDIndexMap map[string]int
	ParityCIDs      [][]string
}

type Client struct {
	*ipfsconnector.IPFSConnector
	IPFSClusterConnector *ipfscluster.Connector
}

// create client
func NewClient(clusterHost string, clusterPort int, ipfsHost string, ipfsPort int) (client *Client, err error) {
	client = &Client{}
	err = client.InitIPFSConnector(ipfsPort, ipfsHost)
	if err != nil {
		return nil, err
	}
	err = client.InitIPFSClusterConnector(clusterPort, clusterHost)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// init ipfs connector for future usage
func (c *Client) InitIPFSConnector(port int, host string) error {
	conn, err := ipfsconnector.CreateIPFSConnector(port, host)
	if err != nil {
		return xerrors.Errorf("fail to connect to IPFS: %s", err)
	}
	c.IPFSConnector = conn

	return nil
}

// init ipfs cluster connector for future usage
func (c *Client) InitIPFSClusterConnector(port int, host string) error {
	conn, err := ipfscluster.CreateIPFSClusterConnector(port, host)
	if err != nil {
		return xerrors.Errorf("fail to connect to IPFS Cluster: %s", err)
	}
	c.IPFSClusterConnector = conn

	return nil
}

// AddAndPinAsFile adds a file to IPFS network and pin the file in cluster with a replication factor
// replicate = 0 means use default config in the cluster
func (c *Client) AddAndPinAsFile(data []byte, replicate int) (cid string, err error) {
	// upload file to IPFS network
	cid, err = c.AddFileFromMem(data)
	if err != nil {
		return "", err
	}

	// pin file in cluster
	err = c.IPFSClusterConnector.AddPin(cid, replicate)
	return cid, err
}

// AddAndPinAsRaw adds raw data to IPFS network and pin it in cluster with a replication factor
// replicate = 0 means use default config in the cluster
func (c *Client) AddAndPinAsRaw(data []byte, replicate int) (cid string, err error) {
	// upload raw bytes to IPFS network
	cid, err = c.AddRawData(data)
	if err != nil {
		return "", err
	}

	// pin data in cluster
	err = c.IPFSClusterConnector.AddPin(cid, replicate)
	return cid, err
}

// GetMetaData downloads metafile from IPFS network and returns a metafile object
func (c *Client) GetMetaData(cid string) (metadata *Metadata, err error) {
	data, err := c.GetFileToMem(cid)
	if err != nil {
		return nil, err
	}
	var myMetadata Metadata
	err = json.Unmarshal(data, &myMetadata)
	if err != nil {
		return nil, err
	}

	return &myMetadata, nil
}
