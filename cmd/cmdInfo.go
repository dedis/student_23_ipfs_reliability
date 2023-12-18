package cmd

import (
	"fmt"
	"io"
	"ipfs-alpha-entanglement-code/Server"
	"ipfs-alpha-entanglement-code/client"
	"ipfs-alpha-entanglement-code/performance"
	"ipfs-alpha-entanglement-code/util"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type Command struct {
	*cobra.Command
	*client.Client
	*Server.Server
}

// NewClient creates a new client for futhur use
func NewCommand() (command *Command, err error) {
	command = &Command{}
	command.initCmd()
	cl, _ := client.NewClient("", 0, "", 0)
	// if err != nil {
	// 	return nil, err
	// }
	command.Client = cl
	command.Server = &Server.Server{}

	return command, nil
}

// initCmd inits cmd for user interaction
func (c *Command) initCmd() {
	c.Command = &cobra.Command{
		Use: "entangler",
	}

	c.AddUploadCmd()
	c.AddDownloadCmd()
	c.AddPerformanceCmd()
	c.AddDaemonCmd()
}

func (c *Command) AddDaemonCmd() {
	var port int
	var communityIP string
	var clusterIP string
	var clusterPort int
	var IpfsIP string
	var IpfsPort int
	var discovery string

	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Start the community node",
		Long:  "Start the community node",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			util.EnableLogPrint()
			util.EnableInfoPrint()

			c.RunServer(port, communityIP, clusterIP, clusterPort, IpfsIP, IpfsPort, discovery)
		},
	}
	daemonCmd.Flags().IntVarP(&port, "port", "p", 7070, "Set the port for corresponding community node")
	daemonCmd.Flags().StringVarP(&communityIP, "community-ip", "i", "localhost", "Sets the IP address of the community node")
	daemonCmd.Flags().StringVarP(&clusterIP, "cluster-ip", "c", "localhost", "Sets the IP address of the cluster node")
	daemonCmd.Flags().IntVarP(&clusterPort, "cluster-port", "v", 9094, "Sets the port of the cluster node")
	daemonCmd.Flags().StringVarP(&IpfsIP, "ipfs-ip", "j", "localhost", "Sets the IP address of the IPFS node")
	daemonCmd.Flags().IntVarP(&IpfsPort, "ipfs-port", "b", 5001, "Sets the port of the IPFS node")
	daemonCmd.Flags().StringVarP(&discovery, "discovery", "d", "http://localhost:3000", "Sets the discovery server address with port")
	c.AddCommand(daemonCmd)
}

// AddUploadCmd enables upload functionality
func (c *Command) AddUploadCmd() {
	var alpha, s, p, replication int
	var cNAddress string
	uploadCmd := &cobra.Command{
		Use:   "upload [path]",
		Short: "Upload a file to IPFS",
		Long:  "Upload a file to IPFS with optional entanglement",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			util.EnableLogPrint()

			cid, metaCID, pinResult, err := c.Upload(args[0], alpha, s, p, replication, cNAddress)
			if len(cid) > 0 {
				log.Println("Finish adding file to IPFS. File CID: ", cid)
			}
			if len(metaCID) > 0 {
				log.Println("Finish adding metaData to IPFS. MetaFile CID: ", metaCID)
			}
			if err != nil {
				log.Println("Error:", err)
				os.Exit(1)
			}
			if pinResult != nil {
				err = pinResult()
				if err != nil {
					log.Println("Error:", err)
					os.Exit(1)
				}
			}
			log.Println("Upload succeeds.")
		},
	}
	uploadCmd.Flags().IntVarP(&alpha, "alpha", "a", 0, "Set entanglement alpha. 0 means no entanglement")
	uploadCmd.Flags().IntVarP(&s, "s", "s", 0, "Set entanglement s")
	uploadCmd.Flags().IntVarP(&p, "p", "p", 0, "Set entanglement p")
	uploadCmd.Flags().IntVarP(&replication, "replication", "r", 5, "Set replication factor for intermediate nodes of EMTs")
	uploadCmd.Flags().StringVarP(&cNAddress, "address", "d", "", "Pass the Community node address:port for monitoring")

	c.AddCommand(uploadCmd)
}

// Define structures to represent the JSON response
type SuccessResponse struct {
	Message string `json:"message"`
	Out     string `json:"out"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func (c *Command) downloadFile(communityAddress, rootFileCID, metadataCID, path string, uploadRecoverData bool, depth int) (string, error) {
	baseURL := fmt.Sprintf("http://%s/downloadFile", communityAddress)

	// Build the query parameters
	params := url.Values{}
	params.Add("rootFileCID", rootFileCID)
	params.Add("metadataCID", metadataCID)
	params.Add("path", path)
	params.Add("uploadRecoverData", fmt.Sprintf("%v", uploadRecoverData))
	params.Add("depth", fmt.Sprintf("%d", depth))

	// Construct the final URL with query parameters
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Send the GET request
	resp, err := http.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Handle the response based on the status code
	switch resp.StatusCode {
	case 200:
		// write file from []byte body to path
		// write file
		out, err := client.WriteFile(rootFileCID, path, body)
		if err != nil {
			return "", fmt.Errorf("error writing file: %v", err)
		}

		return out, nil
	case 400:
		return "", fmt.Errorf("error: %s", string(body))
	default:
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// AddDownloadCmd enables download functionality
func (c *Command) AddDownloadCmd() {
	var opt client.DownloadOption
	var path string
	var communityAddress string
	var depth int
	downloadCmd := &cobra.Command{
		Use:   "download [cid] [path]",
		Short: "Download a file from IPFS",
		Long:  "Download a file from IPFS. Do recovery if data is missing",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			util.EnableLogPrint()

			// send get request to 0.0.0.0:port/downloadFile
			out, err := c.downloadFile(communityAddress, args[0], opt.MetaCID, path, opt.UploadRecoverData, depth)
			if err != nil {
				log.Println("Error:", err)
				os.Exit(1)
			}
			log.Printf("Download succeeds to '%s'.\n", out)
		},
	}
	downloadCmd.Flags().StringVarP(&path, "output", "o", "",
		"Provide output path to store the downloaded stuff")
	downloadCmd.Flags().StringVarP(&opt.MetaCID, "metacid", "m",
		"", "Provide metafile cid for recovery")
	downloadCmd.Flags().StringVarP(&communityAddress, "address", "a", "localhost:7070", "Set the complete address (ip:port) for corresponding community node")

	downloadCmd.Flags().BoolVarP(&opt.UploadRecoverData, "upload-recovery",
		"u", true, "Allow upload recovered chunk back to IPFS network")
	downloadCmd.Flags().IntSliceVar(&opt.DataFilter, "missing-data",
		[]int{}, "Specify the missing data blocks for testing")

	downloadCmd.Flags().IntVarP(&depth, "depth", "d", 1, "Set the depth for repairing the missing data (1 no repair)")

	c.AddCommand(downloadCmd)
}

func (c *Command) AddPerformanceCmd() {
	var rootCmd = &cobra.Command{Use: "perf"}

	var fileCase string
	var lossPercent float32
	var iteration int
	recoverCmd := &cobra.Command{
		Use:   "recover [testcase] [loss-percentage]",
		Short: "Performance test for block recovery",
		Long:  "Performance test for block recovery during download from IPFS",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			util.DisableLogPrint()
			util.DisableInfoPrint()

			rand.Seed(time.Now().UnixNano())
			result := performance.PerfRecovery(fileCase, lossPercent, iteration)
			if result.Err != nil {
				log.Println("Error:", result.Err)
				return
			}
			log.Printf("Data Recovery Rate: %f\n", result.RecoverRate)
			log.Printf("Parity Overhead: %f\n", result.DownloadParity)
			log.Printf("Successfully Downloaded Block: %d\n", result.PartialSuccessCnt)
		},
	}
	recoverCmd.Flags().StringVarP(&fileCase, "testcase", "t", "25MB", "Test cases of different file sizes")
	recoverCmd.Flags().Float32VarP(&lossPercent, "loss-percent", "p", 0.5, "Loss percentage of the parities")
	recoverCmd.Flags().IntVarP(&iteration, "iteration", "i", 5, "Repeat the performance test for several times")
	rootCmd.AddCommand(recoverCmd)

	var repFactor int
	repCmd := &cobra.Command{
		Use:   "rep [testcase] [loss-percentage]",
		Short: "Performance test for blocks replication",
		Long:  "Performance test for blocks that are replicated in the IPFS",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			rand.Seed(time.Now().UnixNano())
			result := performance.PerfReplication(fileCase, lossPercent, repFactor, iteration)
			if result.Err != nil {
				log.Println("Error:", result.Err)
				return
			}
			log.Printf("Data Recovery Rate: %f\n", result.RecoverRate)
			log.Printf("Successfully Downloaded Block: %d\n", result.PartialSuccessCnt)
		},
	}
	repCmd.Flags().StringVarP(&fileCase, "testcase", "t", "25MB", "Test cases of different file sizes")
	repCmd.Flags().Float32VarP(&lossPercent, "loss-percent", "p", 0.5, "Loss percentage of the replication")
	repCmd.Flags().IntVarP(&iteration, "iteration", "i", 5, "Repeat the performance test for several times")
	repCmd.Flags().IntVarP(&repFactor, "rep-factor", "r", 3, "Set the replication factor of the data")
	rootCmd.AddCommand(repCmd)

	c.AddCommand(rootCmd)
}
