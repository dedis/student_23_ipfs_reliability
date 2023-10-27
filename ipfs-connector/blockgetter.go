package ipfsconnector

import (
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"

	"golang.org/x/xerrors"
)

type IPFSGetter struct {
	entangler.BlockGetter
	*IPFSConnector
	DataIndexCIDMap util.SafeMap
	DataFilter      map[int]struct{}
	Parity          [][]string
	ParityFilter    []map[int]struct{}
	BlockNum        int
}

// TODO:
// 1. save tree depth and max children for parity trees in the metadata
// 2. store this info + other metadata info here
// 3. store empty tree here and use it to guide through the recovery process
// 4. when lattice requests a chunk using the index, we traverse this tree get the path to this chunk
// 5. use the path to recursively get their CIDs and store them here
// 6. use the CIDs to get the data and return it to lattice
// 7. we need to store the parity tree root CIDs here
// 8. when a parity is requested, we traverse the parity tree to get the path to this parity
// 9. use the path to recursively get their CIDs and store them here
// 10. use the CIDs to get the data and return it to lattice
// 11. In case we can't find any of the blocks, should we try with MAX_DEPTH to request the chunks from lattice?
// this would actually make another function call, that would come back here,
// will this keep happening until the root or something else??

func CreateIPFSGetter(connector *IPFSConnector, CIDIndexMap map[string]int, parityCIDs [][]string) *IPFSGetter {
	indexToDataCIDMap := *util.NewSafeMap()
	indexToDataCIDMap.AddReverseMap(CIDIndexMap)
	return &IPFSGetter{
		IPFSConnector:   connector,
		DataIndexCIDMap: indexToDataCIDMap,
		Parity:          parityCIDs,
		BlockNum:        len(CIDIndexMap),
	}
}

func (getter *IPFSGetter) GetData(index int) ([]byte, error) {
	/* Get the target CID of the block */
	cid, ok := getter.DataIndexCIDMap.Get(index)
	if !ok {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}

	/* get the data, mask to represent the data loss */
	if getter.DataFilter != nil {
		if _, ok = getter.DataFilter[index]; ok {
			err := xerrors.Errorf("no data exists")
			return nil, err
		}
	}
	data, err := getter.GetRawBlock(cid)
	return data, err

}

func (getter *IPFSGetter) GetParity(index int, strand int) ([]byte, error) {
	if index < 1 || index > getter.BlockNum {
		err := xerrors.Errorf("invalid index")
		return nil, err
	}
	if strand < 0 || strand > len(getter.Parity) {
		err := xerrors.Errorf("invalid strand")
		return nil, err
	}

	/* Get the target CID of the block */
	cid := getter.Parity[strand][index-1]

	/* Get the parity, mask to represent the parity loss */
	if getter.ParityFilter != nil && len(getter.ParityFilter) > strand && getter.ParityFilter[strand] != nil {
		if _, ok := getter.ParityFilter[strand][index]; ok {
			err := xerrors.Errorf("no parity exists")
			return nil, err
		}
	}

	data, err := getter.GetFileToMem(cid)
	return data, err

}
