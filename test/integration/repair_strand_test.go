package integration

import (
	"github.com/stretchr/testify/require"
	"ipfs-alpha-entanglement-code/client"
	"ipfs-alpha-entanglement-code/performance"
	"testing"
)

func Test_Repair_Strand(t *testing.T) {
	// util.EnableLogPrint()
	repair := func(fileCID string, metaCID string, strand int) func(*testing.T) {
		return func(t *testing.T) {
			clientVar, err := client.NewClient()
			require.NoError(t, err)

			err = clientVar.RepairStrand(fileCID, metaCID, strand)
			require.NoError(t, err)
		}
	}

	// for _, testcase := range []string{"5MB", "10MB", "20MB", "25MB"} {
	for _, testcase := range []string{"25MB"} {
		info := performance.InfoMap[testcase]
		// We'll always have at least one strand so we're testing on the first strand
		t.Run(testcase, repair(info.FileCID, info.MetaCID, 1))
	}
}
