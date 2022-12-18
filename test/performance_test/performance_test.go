package test

import (
	"encoding/json"
	"fmt"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"ipfs-alpha-entanglement-code/performance"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Only_Data_Loss(t *testing.T) {
	var allRates []string
	var allOverhead []string
	var accuRate float32
	var accuOverhead float32

	onlyData := func(missNum int, fileinfo performance.FileInfo, try int) func(*testing.T) {
		return func(*testing.T) {
			conn, err := ipfsconnector.CreateIPFSConnector(0)
			require.NoError(t, err)

			// download metafile
			data, err := conn.GetFileToMem(fileinfo.MetaCID)
			require.NoError(t, err)
			var metaData performance.Metadata
			err = json.Unmarshal(data, &metaData)
			require.NoError(t, err)

			// create getter
			getter, err := performance.CreateRecoverGetter(conn, metaData.DataCIDIndexMap, metaData.ParityCIDs)
			require.NoError(t, err)

			for i := 0; i < try; i++ {
				indexes := make([]int, fileinfo.TotalBlock)
				for j := 0; j < fileinfo.TotalBlock; j++ {
					indexes[j] = j + 1
				}
				missedIndexes := map[int]struct{}{}
				for j := 0; j < missNum; j++ {
					r := int(rand.Int63n(int64(len(indexes))))
					missedIndexes[indexes[r]] = struct{}{}
					indexes[r], indexes[len(indexes)-1] = indexes[len(indexes)-1], indexes[r]
					indexes = indexes[:len(indexes)-1]
				}
				getter.DataFilter = missedIndexes

				result := performance.Recovery(fileinfo, metaData, getter)
				accuRate += result.RecoverRate
				accuOverhead += result.DownloadParity
			}

		}
	}

	key := "25MB"
	try := 100
	rand.Seed(time.Now().UnixNano())
	for i := 0; i <= performance.InfoMap[key].TotalBlock; i++ {
		accuRate = 0
		accuOverhead = 0
		t.Run(fmt.Sprintf("test_%d", i), onlyData(i, performance.InfoMap[key], try))

		allRates = append(allRates, fmt.Sprintf("%.4f", accuRate/float32(try)))
		allOverhead = append(allOverhead, fmt.Sprintf("%.4f", float32(accuOverhead)/(float32(try))))
	}

	// 25MB
	// Success Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000]
	// Overhead: [0.0000,1.9500,3.8900,5.8000,7.6800,9.6100,11.5200,13.3000,15.0400,16.7800,18.6200,20.1700,22.2100,23.8700,25.5900,27.4100,29.0200,30.6200,32.3600,33.8400,35.3100,37.0200,38.5000,40.0700,41.7000,43.2500,44.1300,46.0200,47.3600,49.0400,50.3500,51.6200,53.0100,54.6500,55.6700,56.4400,58.2700,59.9000,60.3800,61.9500,63.1400,64.5400,65.2600,66.3900,68.0000,68.2300,70.4200,71.1100,72.4000,73.3400,74.4000,75.2700,75.8300,76.8900,78.6100,79.0700,80.3800,80.9300,82.1700,82.5800,83.5800,84.8300,84.9900,85.7900,86.5500,87.3600,88.0900,89.1100,89.5700,90.0100,90.7900,91.0200,91.7400,92.6200,92.6700,93.8400,93.9600,94.3500,94.8900,95.5400,96.0400,96.2800,96.6000,97.1400,97.5000,97.9200,98.1700,98.3700,98.8400,99.3400,99.4200,99.6800,99.8600,100.0600,100.1800,100.4700,100.5300,100.6500,100.8100,100.8700,100.9800,101.0000]
	fmt.Println("Success Rate: [" + strings.Join(allRates, ",") + "]")
	fmt.Println("Overhead: [" + strings.Join(allOverhead, ",") + "]")
}

func Test_Only_Parity_Loss(t *testing.T) {
	var partialRates []string
	var fullRates []string
	var allOverhead []string

	onlyParity := func(missNum int, fileinfo performance.FileInfo, iteration int) func(*testing.T) {
		return func(*testing.T) {
			result := performance.RecoverWithFilter(fileinfo, missNum, iteration, 0)
			partialRates = append(partialRates, fmt.Sprintf("%.4f", result.RecoverRate))
			fullRates = append(fullRates, fmt.Sprintf("%.4f", result.FullSuccessCnt))
			allOverhead = append(allOverhead, fmt.Sprintf("%.4f", result.DownloadParity))
		}
	}

	key := "75MB"
	try := 100
	rand.Seed(time.Now().UnixNano())
	for i := 558; i <= performance.InfoMap[key].TotalBlock*3; i += 3 {
		t.Run(fmt.Sprintf("test_%d", i), onlyParity(i, performance.InfoMap[key], try))
		fmt.Println("Success Partial Recovery Rate: [" + strings.Join(partialRates, ",") + "]")
		fmt.Println("Success Full Recovery Rate: [" + strings.Join(fullRates, ",") + "]")
		fmt.Println("Overhead: [" + strings.Join(allOverhead, ",") + "]")
	}

	// 5MB
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9995,1.0000,0.9990,0.9990,0.9581,0.9686,0.9152,0.9448,0.9352,0.8438,0.8267,0.8386,0.7148,0.7614,0.5838,0.5919,0.4257,0.3933,0.3886,0.3081,0.2514,0.1262,0.1586,0.0633,0.0833,0.0471,0.0305,0.0400,0.0210,0.0186,0.0152,0.0171,0.0090,0.0067,0.0024,0.0019,0.0014,0.0014,0.0000,0.0000,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,0.9800,0.9800,0.9200,0.9400,0.8400,0.8500,0.8600,0.7700,0.6600,0.6900,0.4500,0.5200,0.2300,0.2200,0.1400,0.0500,0.0100,0.0100,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [21.0000,21.6600,22.4300,22.9700,23.4700,24.2300,24.8800,25.4900,27.9300,28.7100,31.1700,32.0300,33.4500,34.8500,36.2700,38.8600,39.8300,39.7700,43.4800,43.8700,44.8300,46.1600,48.0300,47.8200,48.1800,47.9400,48.4200,48.4100,48.5400,47.3400,48.0200,47.5200,46.1000,45.3900,44.2500,41.7500,42.6700,38.6700,38.6900,33.5800,31.6100,30.6400,27.6800,25.6100,22.2700,21.1400,19.0100,17.8200,16.0800,14.8600,13.7100,12.5600,11.3900,10.3300,9.1500,8.1700,7.1300,6.0300,5.0000,4.0100,3.0000,2.0000,1.0000,0.0000]

	// 25MB
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9999,1.0000,1.0000,1.0000,1.0000,1.0000,0.9999,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9897,0.9900,0.9999,0.9998,0.9900,0.9997,1.0000,0.9997,0.9999,0.9998,0.9898,1.0000,0.9997,0.9999,0.9898,0.9997,0.9799,0.9899,0.9798,0.9897,0.9894,0.9997,0.9998,1.0000,0.9998,0.9796,0.9894,0.9898,0.9900,0.9899,0.9588,0.9989,0.9684,0.9784,0.9887,0.9685,0.9887,0.9888,0.9775,0.9684,0.9673,0.9556,0.9740,0.9766,0.9554,0.9373,0.9550,0.9246,0.9451,0.9539,0.9323,0.9446,0.9237,0.9230,0.8821,0.8819,0.9294,0.8848,0.8544,0.8907,0.8537,0.8560,0.8613,0.8208,0.7748,0.8843,0.7563,0.8108,0.7401,0.7995,0.8289,0.7634,0.6887,0.7375,0.7648,0.6922,0.6605,0.6305,0.6380,0.6273,0.5539,0.5536,0.5641,0.5438,0.4906,0.5430,0.5111,0.5226,0.4372,0.3892,0.3572,0.3562,0.3977,0.2898,0.3410,0.3508,0.2874,0.2588,0.2882,0.2564,0.2783,0.2409,0.2557,0.1945,0.2237,0.2251,0.1877,0.1913,0.1716,0.1658,0.1537,0.1320,0.1661,0.1074,0.1461,0.0830,0.0850,0.1224,0.1087,0.0766,0.0924,0.0971,0.0877,0.0773,0.0801,0.0757,0.0829,0.0569,0.0524,0.0508,0.0553,0.0436,0.0485,0.0422,0.0296,0.0251,0.0330,0.0293,0.0319,0.0195,0.0375,0.0242,0.0207,0.0190,0.0188,0.0169,0.0171,0.0094,0.0207,0.0121,0.0107,0.0043,0.0104,0.0188,0.0147,0.0086,0.0077,0.0127,0.0076,0.0084,0.0127,0.0051,0.0053,0.0049,0.0065,0.0044,0.0054,0.0038,0.0016,0.0043,0.0029,0.0012,0.0022,0.0016,0.0017,0.0012,0.0021,0.0011,0.0016,0.0000,0.0004,0.0008,0.0000,0.0005,0.0000,0.0003,0.0002,0.0001,0.0000,0.0004,0.0000,0.0001,0.0000,0.0002,0.0003,0.0002,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9600,0.9900,0.9900,0.9900,0.9900,0.9700,1.0000,0.9700,0.9900,0.9800,0.9700,1.0000,0.9700,0.9900,0.9700,0.9700,0.9700,0.9800,0.9600,0.9600,0.9600,0.9700,0.9800,1.0000,0.9800,0.9400,0.9300,0.9700,0.9900,0.9800,0.8700,0.9000,0.8500,0.9300,0.8800,0.9000,0.9300,0.8900,0.9000,0.8600,0.8400,0.8300,0.7800,0.7900,0.7700,0.8100,0.8100,0.7400,0.7400,0.7800,0.7100,0.7100,0.6800,0.6700,0.6200,0.6200,0.6300,0.5300,0.5100,0.5500,0.5900,0.4800,0.4000,0.4400,0.3100,0.4800,0.3200,0.2800,0.3000,0.2300,0.2800,0.2100,0.1600,0.1200,0.1500,0.1800,0.0500,0.1000,0.1000,0.0200,0.0400,0.0500,0.0100,0.0100,0.0100,0.0200,0.0200,0.0100,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [101.0000,101.8600,103.0500,103.9800,104.8900,105.7700,106.9000,107.3600,107.9100,109.8600,110.3600,111.5500,111.3800,113.1100,113.9100,115.0100,116.0000,115.7400,117.8100,118.0900,119.8000,120.7700,121.9200,122.6300,124.2300,125.9800,127.4400,126.8200,127.5900,129.7300,131.3600,134.1000,135.0700,134.1400,137.9800,137.4000,140.3900,140.2400,142.4800,143.1100,146.8600,145.7100,150.3600,150.2500,152.4900,154.1300,153.3200,159.9000,162.0800,160.2200,164.2300,165.1200,166.1800,169.8800,170.5900,172.5700,174.4800,178.4300,179.4500,183.3100,180.6100,185.1900,187.5200,190.1900,193.7400,193.8000,197.6800,196.2900,198.6400,202.8200,202.8500,202.9200,207.5000,209.6700,208.6500,210.2300,213.7900,214.8900,215.3600,220.0600,218.4900,220.6800,222.5500,225.3500,226.1300,230.4500,228.5000,231.7600,234.9500,233.8900,233.1200,234.5900,236.0800,237.2300,238.9700,238.3400,241.6400,242.0000,243.1200,244.3000,246.3400,243.5300,248.6300,249.4100,247.5200,249.9000,252.5600,251.5100,254.7200,255.8800,251.6200,258.3600,258.8200,259.4900,255.2500,260.3000,259.4400,261.4900,259.1500,263.5500,261.4900,264.2000,263.0200,265.5400,265.3600,262.6600,265.0600,263.9900,267.8900,268.5800,264.1700,268.9900,266.8500,270.3700,269.2600,268.7400,271.4900,270.6800,270.5000,269.8500,270.5400,269.7000,271.6400,272.5100,271.3900,269.5200,272.0500,269.0300,270.3100,272.8300,270.8700,273.1700,270.8600,272.9000,267.6500,269.0300,272.7500,269.3500,266.8800,268.7100,267.6900,266.9400,268.7800,264.4900,258.6700,267.9400,261.9900,263.4400,258.9600,261.1500,264.6700,256.1000,250.5600,255.3600,256.4200,247.2900,246.2900,242.6800,240.8200,239.6200,231.4300,228.8300,219.7200,227.7300,216.5500,215.9800,213.8100,211.1900,199.8200,194.0600,190.9600,190.8700,188.0000,178.5200,179.7100,174.4200,167.8600,166.0100,164.2800,157.3900,158.1900,153.4600,150.6300,145.5300,143.0600,144.5100,138.2000,137.6600,135.6300,130.5000,128.3700,125.5500,125.8500,120.0900,120.6200,116.7300,113.8200,112.8700,111.0800,108.3100,107.2000,104.7900,103.5200,100.7600,99.1200,97.4300,94.9400,93.7800,91.6500,88.8200,88.0800,87.1700,85.6300,84.3400,81.7500,80.0900,78.4900,76.3800,75.3200,73.6600,72.4700,70.8600,69.8100,68.0500,66.5600,65.3300,63.4100,62.7000,61.5500,59.6400,58.5400,56.7200,56.2200,54.7700,53.2900,51.9400,50.6100,49.4100,48.2700,47.1800,45.9100,44.6800,43.2400,42.4200,41.2100,39.7600,38.8600,37.4700,36.3400,35.3800,34.3100,33.1800,31.9300,30.9500,29.9200,28.6500,27.5300,26.5100,25.4200,24.4600,23.3600,22.4500,21.3500,20.2100,19.2300,18.1400,17.1700,16.1400,15.0800,14.0400,13.0300,12.0400,11.0100,10.0200,9.0000,8.0000,7.0000,6.0000,5.0000,4.0000,3.0000,2.0000,1.0000,0.0000]

	// 75MB
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9942,0.9942,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9999,1.0000,0.9942,1.0000,1.0000,1.0000,1.0000,0.9999,1.0000,0.9999,1.0000,0.9999,0.9999,1.0000,0.9999,1.0000,0.9999,0.9999,0.9942,0.9999,0.9999,0.9883,0.9941,0.9942,0.9883,0.9941,0.9882,0.9992,0.9998,0.9995,0.9697,0.9994,0.9937,0.9897,0.9892,0.9995,0.9991,0.9663,0.9754,0.9824,0.9777,0.9827,0.9664,0.9835,0.9933,0.9713,0.9635,0.9715,0.9765,0.9442,0.9683,0.9559,0.9379,0.9710,0.9617,0.9670,0.9579,0.8924,0.8863,0.9407,0.8887,0.8607,0.9372,0.8546,0.8569,0.9054,0.8481,0.9039,0.7705,0.8057,0.7875,0.7780,0.7949,0.7471,0.7616,0.6700,0.6931,0.6653,0.6652,0.6166,0.6227,0.6414,0.6130,0.5111,0.5161,0.4745,0.4855,0.5269,0.4392,0.4360,0.4490,0.4123,0.3910,0.4279,0.3618,0.3780,0.3128,0.2714,0.1727,0.2501,0.2991,0.2073,0.2179,0.2179,0.2073,0.1803,0.1554,0.1213,0.1142,0.1180,0.1054,0.1174,0.0953,0.1401,0.0864,0.0768,0.0967,0.0784,0.0712,0.0325,0.0581,0.0560,0.0478,0.0310,0.0441,0.0441,0.0184,0.0329,0.0224,0.0221,0.0257,0.0265,0.0363,0.0227,0.0209,0.0111,0.0181,0.0126,0.0121,0.0124,0.0045,0.0105,0.0027,0.0141,0.0070,0.0082,0.0075,0.0058,0.0046,0.0067,0.0029,0.0046,0.0040,0.0039,0.0035,0.0025,0.0020,0.0016,0.0020,0.0032,0.0020,0.0010,0.0019,0.0008,0.0011,0.0025,0.0018,0.0021,0.0001,0.0007,0.0014,0.0002,0.0007,0.0002,0.0001,0.0001,0.0004,0.0011,0.0009,0.0004,0.0001,0.0001,0.0001,0.0000,0.0004,0.0004,0.0001,0.0003,0.0001,0.0001,0.0001,0.0001,0.0000,0.0000,0.0000,0.0001,0.0001,0.0000,0.0000,0.0000,0.0001,0.0000,0.0001,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,0.9900,0.9900,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9800,0.9900,0.9800,1.0000,1.0000,1.0000,0.9900,0.9700,0.9900,0.9800,1.0000,0.9800,0.9600,1.0000,0.9700,0.9900,0.9800,0.9600,0.9700,0.9800,0.9600,0.9300,0.9600,0.9700,0.9500,0.9500,0.9300,0.9000,0.9400,0.9300,0.8700,0.9000,0.9100,0.9200,0.8900,0.8800,0.8900,0.8200,0.7800,0.8200,0.8400,0.7500,0.8400,0.7900,0.8000,0.7400,0.8000,0.7300,0.6800,0.6800,0.6000,0.6700,0.6500,0.5600,0.4300,0.5700,0.4300,0.4000,0.4600,0.4500,0.4200,0.3200,0.3500,0.2200,0.2500,0.2800,0.3000,0.2400,0.2200,0.2200,0.1900,0.1700,0.0700,0.0700,0.0500,0.0900,0.0500,0.0500,0.0100,0.0100,0.0100,0.0000,0.0300,0.0100,0.0200,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [303.0000,305.1500,306.9300,308.9600,310.6500,312.9300,314.5200,317.1300,319.0800,320.6500,322.5300,325.3900,327.0300,329.0400,331.3700,331.9700,334.9200,338.1900,338.8000,342.5000,343.2900,346.0900,347.4100,350.8600,352.9000,355.3200,358.5400,363.3100,365.0100,367.2100,367.0100,370.9300,371.8100,376.3200,377.7700,383.7400,389.2600,387.0200,389.3600,395.3200,395.1800,401.4100,405.7800,408.7300,413.0800,416.4100,416.5700,422.6400,426.4500,429.7900,434.8700,439.3500,440.2800,450.6800,449.7700,453.9800,459.5300,465.2000,470.4500,476.8700,481.6200,487.9200,489.4600,493.2600,496.6100,502.6200,513.5300,514.8000,522.5800,528.5000,538.3300,538.6700,547.2800,553.0800,558.0200,561.7200,568.0300,578.2600,581.3600,586.1500,589.0900,596.6000,598.2900,606.4200,607.7200,620.8000,622.2400,623.1300,627.9800,637.6000,640.6100,641.9500,646.5100,653.2200,662.8500,665.4900,673.0200,675.3200,682.7800,686.3000,691.0400,690.6300,698.4700,699.3300,705.7700,715.8400,711.9900,711.0000,716.7600,719.8600,726.9700,724.5600,725.9300,732.6300,737.0500,735.2400,744.9600,748.5400,748.5700,734.6100,758.1600,754.5600,756.8400,761.9400,771.8100,768.6600,758.5400,768.6100,771.2400,769.5600,773.1100,768.9500,776.4600,789.5100,777.5100,773.9700,782.4500,783.7500,782.5500,788.7200,783.5900,780.8900,799.1100,793.3400,797.7500,790.8900,769.1700,765.9400,789.0300,780.7000,760.4800,794.3400,768.2200,772.3600,788.1300,755.7200,798.9600,746.6400,763.8400,758.7300,757.3300,768.3100,748.8200,759.7100,743.8400,722.1900,712.9700,716.9300,703.1800,731.4400,718.1200,715.7300,685.0100,681.6300,669.7900,699.1400,701.4400,666.0000,671.8300,665.1600,638.2300,622.2900,657.9800,603.2300,639.8900,583.1000,594.1900,561.1400,584.8000,579.3400,537.5000,549.1700,532.0300,542.2100,516.8500,511.6900,494.3900,491.5300,482.0500,473.1600,462.8900,458.2500,456.5900,440.4900,438.4600,434.7300,424.6400,416.7400,402.8200,395.9500,391.4800,385.8100,376.0900,370.4300,362.2600,356.4900,351.3600,346.3400,340.2000,334.8400,325.2600,321.2300,312.6800,308.3200,302.3000,297.2900,290.6200,284.7200,279.7000,273.6800,269.7700,263.5000,260.0900,253.6900,248.7300,242.7300,237.8200,233.9500,228.5200,223.5400,220.5900,215.8100,211.1700,207.4300,201.0700,198.3900,193.5200,188.9600,184.6800,181.2400,177.5400,173.2500,168.5100,165.2200,160.9800,157.2000,153.7000,149.9600,146.6700,142.2800,138.2200,134.4100,131.1500,127.7100,124.4900,120.6400,116.7900,113.9200,109.8300,106.5600,103.0400,99.8900,96.8400,93.8100,90.1600,86.4900,83.4400,80.1500,76.9100,73.5700,70.3100,67.0500,64.1000,60.6500,57.8400,54.6400,51.4500,48.4100,45.4100,42.2700,39.2200,36.1800,33.1200,30.1000,27.0600,24.0600,21.0000,18.0200,15.0100,12.0100,9.0000,6.0000,3.0000,0.0000]

	// 125MB
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9965,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9965,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9999,0.9999,1.0000,0.9998,1.0000,1.0000,0.9965,1.0000,1.0000,0.9999,0.9999,0.9999,0.9999,0.9999,0.9964,0.9964,1.0000,0.9861,0.9997,0.9999,0.9996,0.9996,0.9999,0.9997,0.9925,0.9994,0.9995,0.9962,0.9689,0.9992,0.9962,0.9763,0.9794,0.9919,0.9890,0.9859,0.9720,0.9878,0.9554,0.9877,0.9808,0.9682,0.9572,0.9285,0.9642,0.9522,0.9345,0.9199,0.9794,0.9149,0.9009,0.9322,0.8926,0.8876,0.9044,0.8788,0.8457,0.8474,0.8845,0.8218,0.8104,0.8395,0.8552,0.8578,0.7929,0.7770,0.7581,0.7059,0.6933,0.6790,0.6986,0.7325,0.6721,0.6851,0.6374,0.5850,0.5750,0.5201,0.5197,0.5361,0.5228,0.5089,0.4338,0.3881,0.4085,0.4136,0.3825,0.3669,0.4005,0.3815,0.3090,0.3144,0.2855,0.3186,0.2587,0.2322,0.2112,0.1719,0.2019,0.1358,0.1460,0.1070,0.1397,0.1171,0.1060,0.1016,0.0946,0.0696,0.0607,0.0784,0.0861,0.0574,0.0495,0.0376,0.0641,0.0561,0.0521,0.0506,0.0452,0.0281,0.0337,0.0286,0.0298,0.0298,0.0290,0.0211,0.0086,0.0161,0.0068,0.0130,0.0110,0.0095,0.0138,0.0053,0.0100,0.0083,0.0033,0.0024,0.0095,0.0078,0.0031,0.0048,0.0074,0.0027,0.0015,0.0035,0.0023,0.0035,0.0016,0.0026,0.0009,0.0014,0.0017,0.0013,0.0017,0.0015,0.0015,0.0022,0.0003,0.0013,0.0001,0.0008,0.0005,0.0004,0.0008,0.0001,0.0005,0.0001,0.0001,0.0003,0.0000,0.0002,0.0004,0.0003,0.0001,0.0001,0.0001,0.0002,0.0000,0.0001,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0001,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9900,1.0000,0.9900,1.0000,0.9900,0.9900,0.9900,1.0000,0.9900,0.9900,1.0000,0.9900,1.0000,0.9900,1.0000,0.9900,0.9900,0.9900,0.9900,0.9700,0.9600,1.0000,0.9800,0.9900,0.9800,0.9700,0.9900,0.9800,0.9500,0.9500,0.9700,0.9300,0.9700,0.9500,0.9500,1.0000,0.8800,0.9100,0.9300,0.8800,0.9200,0.9600,0.9100,0.9100,0.8800,0.8500,0.8200,0.7900,0.8000,0.8300,0.8600,0.8000,0.7500,0.8100,0.7500,0.6900,0.7000,0.6200,0.5600,0.6200,0.6400,0.5700,0.5900,0.5600,0.4100,0.3900,0.3900,0.4300,0.2900,0.3200,0.2100,0.1800,0.2800,0.2400,0.2100,0.2300,0.1900,0.1600,0.1100,0.0800,0.1100,0.0400,0.0800,0.0400,0.0400,0.0400,0.0200,0.0000,0.0100,0.0000,0.0100,0.0000,0.0000,0.0000,0.0100,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [504.0000,508.2300,510.3600,513.5800,516.1800,520.5100,523.9000,527.3000,530.6300,535.0200,537.9500,541.4900,544.4300,548.0100,550.0500,555.5400,559.1300,563.7000,565.3000,568.7800,572.3800,577.7700,580.0100,586.7200,589.2700,594.8900,600.6800,602.6200,606.3900,611.0800,616.1800,620.4500,622.8100,627.3300,632.0100,641.7500,646.1900,651.5700,656.6300,657.7400,661.0900,669.2300,671.5700,676.7100,684.4700,697.1800,696.0600,704.4100,710.3500,718.6100,729.9600,732.1400,748.5200,747.9900,760.3700,757.8000,768.5900,784.2600,787.7100,791.9900,797.5500,803.2400,816.1900,824.6800,830.4200,846.5900,855.0400,864.5500,869.4700,880.0900,881.1500,904.1100,908.6800,922.1400,929.0400,938.9500,941.5500,956.6600,970.2400,982.1700,980.8200,995.4700,999.9600,1015.8400,1015.9100,1025.3500,1032.6801,1047.9100,1053.1000,1061.6300,1076.4100,1075.1400,1087.8400,1093.3300,1102.4000,1115.6600,1119.4500,1122.6200,1129.9301,1140.2200,1142.4200,1149.8800,1164.2800,1162.8300,1179.2000,1177.1200,1193.1100,1195.9000,1199.7100,1198.3400,1213.6100,1217.9500,1208.4600,1231.5200,1233.7800,1243.3300,1236.5900,1253.1100,1255.7700,1255.2400,1268.2700,1274.0000,1272.9500,1248.8400,1284.7500,1295.2500,1282.7500,1285.3700,1297.7800,1308.1300,1299.8000,1293.7400,1316.5699,1280.0100,1337.1801,1337.7100,1325.0500,1319.1000,1285.8700,1326.7400,1331.1300,1314.2800,1303.3101,1359.5601,1314.0100,1313.1500,1337.1700,1285.3600,1309.8700,1300.0900,1293.2900,1276.4600,1287.4800,1306.5800,1249.0000,1279.7800,1292.6300,1299.3600,1270.1300,1274.9700,1256.3400,1230.8500,1210.9301,1173.6801,1164.0200,1191.8400,1204.0000,1192.0800,1175.3700,1178.9000,1117.1000,1122.0300,1106.5400,1092.0400,1096.8300,1100.2100,1071.5601,1022.7600,1039.6200,1044.1100,1046.3900,1006.2100,998.2400,1002.8300,1009.1700,958.2800,958.6300,902.0400,963.3700,857.6800,932.2600,846.4700,808.8200,866.9800,779.8500,804.2600,771.8700,775.9500,742.6100,743.6100,739.4400,729.5600,724.2800,710.9600,691.9000,693.2300,660.9100,663.4400,653.1200,661.4200,642.9600,630.1100,624.0700,603.1500,596.0300,585.7700,569.7000,559.4400,556.7700,546.3600,536.1100,527.5700,517.5300,508.9500,497.8100,490.4200,483.5800,470.4200,462.1400,454.5300,443.6800,433.1400,427.6900,420.3400,411.2800,403.0700,390.0900,387.9800,379.4700,368.9000,362.1300,357.4900,347.5000,341.2400,335.0300,325.6100,319.8000,312.1600,305.2600,297.7300,291.7000,284.6400,277.9100,272.6200,264.8400,258.1800,251.9500,245.2100,239.7000,233.9500,227.0800,221.2800,215.4200,210.1300,203.1700,197.7900,191.5900,186.1000,180.1400,174.3900,168.9700,162.8200,158.2100,152.0100,146.5600,140.9500,135.4700,130.2800,124.7700,119.5800,114.3300,109.0400,103.8500,98.3700,93.1200,88.0800,82.9600,77.8200,72.4700,67.4700,62.2800,57.3400,52.2400,47.1700,42.1300,37.0400,32.0200,27.0300,22.0000,17.0000,12.0000,7.0000,2.0000,0.0000]
	fmt.Println("Success Partial Recovery Rate: [" + strings.Join(partialRates, ",") + "]")
	fmt.Println("Success Full Recovery Rate: [" + strings.Join(fullRates, ",") + "]")
	fmt.Println("Overhead: [" + strings.Join(allOverhead, ",") + "]")
}

func Test_Only_Parity_Loss_Node_Loss(t *testing.T) {
	var partialRates []string
	var fullRates []string
	var allOverhead []string
	nbNodes := 100

	onlyParity := func(missNum int, fileinfo performance.FileInfo, iteration int) func(*testing.T) {
		return func(*testing.T) {
			result := performance.RecoverWithFilter(fileinfo, missNum, iteration, nbNodes)
			partialRates = append(partialRates, fmt.Sprintf("%.4f", result.RecoverRate))
			fullRates = append(fullRates, fmt.Sprintf("%.4f", result.FullSuccessCnt))
			allOverhead = append(allOverhead, fmt.Sprintf("%.4f", result.DownloadParity))
		}
	}

	key := "25MB"
	try := 500
	rand.Seed(time.Now().UnixNano())
	for i := 0; i <= nbNodes; i++ {
		t.Run(fmt.Sprintf("test_%d", i), onlyParity(i, performance.InfoMap[key], try))
	}

	// Node = 5
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,0.9165,0.1198,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,0.4520,0.0000,0.0000]
	// Overhead: [101.0000,138.2000,220.7800,227.9660,79.3360,0.0000]

	// Node = 10
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,0.9999,0.9997,0.9971,0.8834,0.2716,0.0564,0.0031,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,0.9940,0.9700,0.9100,0.3840,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [101.0000,119.6180,143.8900,187.1860,229.9180,252.1760,246.6680,122.7520,64.6820,30.3160,0.0000]

	// Node = 20
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,1.0000,0.9999,1.0000,0.9997,0.9997,0.9975,0.9933,0.9843,0.9430,0.8210,0.5219,0.2416,0.1209,0.0586,0.0242,0.0079,0.0015,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,1.0000,0.9940,0.9980,0.9660,0.9660,0.9440,0.9280,0.8400,0.6340,0.2880,0.0100,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [101.0000,110.1080,119.5280,130.7960,146.8500,165.4800,196.8920,218.6420,240.7340,253.7420,259.7820,263.4640,251.6500,182.6140,118.1520,87.7700,64.9600,46.1760,30.3080,15.1540,0.0000]

	// Node = 50
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9999,1.0000,0.9999,0.9999,0.9999,0.9998,0.9999,0.9997,0.9997,0.9996,0.9994,0.9935,0.9974,0.9991,0.9928,0.9922,0.9880,0.9772,0.9691,0.9611,0.9105,0.8817,0.8187,0.6871,0.5142,0.4055,0.2984,0.2342,0.1950,0.1534,0.1164,0.0908,0.0637,0.0428,0.0368,0.0228,0.0181,0.0094,0.0072,0.0028,0.0015,0.0006,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9980,1.0000,0.9920,0.9960,0.9920,0.9940,0.9880,0.9800,0.9880,0.9720,0.9720,0.9640,0.9420,0.9460,0.9380,0.9140,0.8740,0.8420,0.8140,0.7840,0.6720,0.5900,0.4160,0.2920,0.1620,0.0380,0.0080,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [101.0000,104.6140,108.3840,112.2540,116.1380,120.6380,125.2960,130.6340,135.9500,143.0300,149.8680,157.7780,170.3300,182.1760,192.8660,207.9240,217.2000,226.8620,234.7640,241.7020,247.2940,250.5500,256.0560,258.9340,261.0480,262.2400,266.0200,266.5740,264.3560,261.1880,251.2040,224.3840,185.1660,156.4340,135.8500,120.3300,106.8240,96.0040,85.3500,75.5740,67.0920,58.8600,51.7420,44.3740,37.5240,30.7760,24.4780,18.2180,12.1160,6.0580,0.0000]

	// Node = 100
	// Success Partial Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9999,0.9999,0.9999,0.9999,0.9998,0.9999,0.9998,0.9999,0.9996,0.9998,0.9998,0.9976,0.9957,0.9996,0.9977,0.9996,0.9996,0.9972,0.9993,0.9955,0.9973,0.9990,0.9987,0.9947,0.9944,0.9903,0.9882,0.9852,0.9926,0.9847,0.9797,0.9816,0.9578,0.9584,0.9650,0.9421,0.9160,0.8884,0.8771,0.8176,0.8335,0.7647,0.6990,0.6099,0.5560,0.4996,0.4340,0.3876,0.3555,0.3092,0.2763,0.2528,0.2214,0.1961,0.1838,0.1599,0.1397,0.1161,0.1088,0.0894,0.0803,0.0732,0.0628,0.0525,0.0381,0.0373,0.0323,0.0257,0.0207,0.0174,0.0148,0.0110,0.0091,0.0074,0.0047,0.0032,0.0020,0.0014,0.0012,0.0005,0.0002,0.0000]
	// Success Full Recovery Rate: [1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,1.0000,0.9960,1.0000,1.0000,0.9980,1.0000,0.9980,0.9960,1.0000,0.9980,0.9960,0.9940,0.9920,0.9900,0.9860,0.9820,0.9940,0.9840,0.9860,0.9640,0.9800,0.9840,0.9620,0.9660,0.9620,0.9720,0.9560,0.9560,0.9280,0.9320,0.9460,0.9280,0.9060,0.8820,0.8700,0.8740,0.8480,0.8400,0.7860,0.7760,0.7580,0.7320,0.6960,0.6160,0.5820,0.5180,0.4680,0.3940,0.2880,0.2080,0.1580,0.0960,0.0560,0.0160,0.0120,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000,0.0000]
	// Overhead: [101.0000,102.8000,104.6280,106.5880,108.3420,110.3840,112.3320,114.4040,116.4320,119.0100,121.0420,123.2580,125.6220,129.0040,131.4780,134.5280,136.6300,140.7720,144.8820,148.8020,154.3640,158.5400,164.0740,170.4500,174.0700,180.1180,186.1620,193.0600,197.3320,202.0540,207.7260,211.5800,218.7740,221.6540,227.5500,230.5000,234.0100,237.9920,241.1840,245.5660,247.0360,249.9140,251.8280,253.6360,255.4080,256.7220,257.6900,260.6080,261.0560,262.1880,263.3660,263.9620,264.0440,266.1080,265.3580,263.9300,262.1420,259.9320,256.3180,253.6220,243.2040,229.6880,214.1120,197.4180,182.0080,166.5460,156.4100,146.2620,137.8780,129.4100,123.0680,115.3200,109.5860,103.4420,97.5100,92.1100,87.0280,81.7480,77.1160,72.6140,68.1680,64.0420,60.0260,55.8440,52.2220,48.2560,44.7440,41.1980,37.7640,34.3760,30.9800,27.8040,24.6220,21.4240,18.3300,15.2000,12.1300,9.1120,6.0460,3.0240,0.0000]
	fmt.Println("Success Partial Recovery Rate: [" + strings.Join(partialRates, ",") + "]")
	fmt.Println("Success Full Recovery Rate: [" + strings.Join(fullRates, ",") + "]")
	fmt.Println("Overhead: [" + strings.Join(allOverhead, ",") + "]")
}
