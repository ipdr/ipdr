package main

import (
	"errors"
	"fmt"
	"os"

	color "github.com/fatih/color"
	registry "github.com/miguelmota/ipdr/registry"
	regutil "github.com/miguelmota/ipdr/regutil"
	"github.com/miguelmota/ipdr/server"
	log "github.com/sirupsen/logrus"
	cobra "github.com/spf13/cobra"
)

var green = color.New(color.FgGreen)

var (
	// ErrImageIDRequired is error for when image ID is required
	ErrImageIDRequired = errors.New("image hash or name is required")
	// ErrOnlyOneArgumentRequired is error for when one argument only is required
	ErrOnlyOneArgumentRequired = errors.New("only one argument is required")
	// ErrInvalidConvertFormat is error for when convert format is invalid
	ErrInvalidConvertFormat = errors.New("convert format must be either \"docker\" or \"ipfs\"")
)

func main() {
	if os.Getenv("DEBUG") != "" {
		log.SetReportCaller(true)
	}

	var ipfsHost string
	var ipfsGateway string
	var format string
	var dockerRegistryHost string
	var port uint
	var tlsCertPath string
	var tlsKeyPath string
	var silent bool

	rootCmd := &cobra.Command{
		Use:   "ipdr",
		Short: "InterPlanetary Docker Registry",
		Long: `The command-line interface for the InterPlanetary Docker Registry.
More info: https://github.com/miguelmota/ipdr`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Push image to IPFS-backed Docker registry",
		Long:  "Push the Docker image to the InterPlanetary Docker Registry hosted on IPFS",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return ErrImageIDRequired
			}
			if len(args) != 1 {
				return ErrOnlyOneArgumentRequired
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewRegistry(&registry.Config{
				DockerLocalRegistryHost: dockerRegistryHost,
				IPFSHost:                ipfsHost,
				IPFSGateway:             ipfsGateway,
				Debug:                   !silent,
			})

			imageID := args[0]

			hash, err := reg.PushImageByID(imageID)
			if err != nil {
				return err
			}

			if silent {
				fmt.Println(hash)
			} else {
				fmt.Println(green.Sprintf("\nSuccessfully pushed Docker image to IPFS:\n/ipfs/%s", hash))
			}
			return nil
		},
	}

	pushCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Silent flag suppresses logs and outputs only IPFS hash")
	pushCmd.Flags().StringVarP(&ipfsHost, "ipfs-host", "", "127.0.0.1:5001", "A remote IPFS API host to push the image to. Eg. 127.0.0.1:5001")
	pushCmd.Flags().StringVarP(&dockerRegistryHost, "docker-registry-host", "", "docker.localhost:5000", "The Docker local registry host. Eg. 127.0.0.1:5000 Eg. docker.localhost:5000")

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull image from the IPFS-backed Docker registry",
		Long:  "Pull the Docker image from the InterPlanetary Docker Registry hosted on IPFS",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return ErrImageIDRequired
			}
			if len(args) != 1 {
				return ErrOnlyOneArgumentRequired
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewRegistry(&registry.Config{
				DockerLocalRegistryHost: dockerRegistryHost,
				IPFSHost:                ipfsHost,
				IPFSGateway:             ipfsGateway,
				Debug:                   !silent,
			})

			imageHash := args[0]
			tag, err := reg.PullImage(imageHash)
			if err != nil {
				return err
			}

			if silent {
				fmt.Println(tag)
			} else {
				fmt.Println(green.Sprintf("\nSuccessfully pulled Docker image from IPFS:\n%s", tag))
			}
			return nil
		},
	}

	pullCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Silent flag suppresses logs and outputs only Docker repo tag")
	pullCmd.Flags().StringVarP(&ipfsHost, "ipfs-host", "", "127.0.0.1:5001", "A remote IPFS API host to pull the image from. Eg. 127.0.0.1:5001")
	pullCmd.Flags().StringVarP(&ipfsGateway, "ipfs-gateway", "g", "127.0.0.1:8080", "The readonly IPFS Gateway URL to pull the image from. Eg. https://ipfs.io")
	pullCmd.Flags().StringVarP(&dockerRegistryHost, "docker-registry-host", "", "docker.localhost:5000", "The Docker local registry host. Eg. 127.0.0.1:5000 Eg. docker.localhost:5000")

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start IPFS-backed Docker registry server",
		Long:  "Start the IPFS-backed Docker registry server that proxies images stored on IPFS to Docker registry format",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.NewServer(&server.Config{
				Port:        port,
				Debug:       !silent,
				IPFSGateway: ipfsGateway,
				TLSKeyPath:  tlsKeyPath,
				TLSCertPath: tlsCertPath,
			})

			return srv.Start()
		},
	}

	serverCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Silent flag suppresses logs")
	serverCmd.Flags().UintVarP(&port, "port", "p", 5000, "The port for the Docker registry to listen on")
	serverCmd.Flags().StringVarP(&tlsCertPath, "tlsCertPath", "", "", "The path to the .crt file for TLS")
	serverCmd.Flags().StringVarP(&tlsKeyPath, "tlsKeyPath", "", "", "The path to the .key file for TLS")
	serverCmd.Flags().StringVarP(&ipfsGateway, "ipfs-gateway", "g", "127.0.0.1:8080", "The readonly IPFS Gateway URL to pull the image from. Eg. https://ipfs.io")

	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert a hash to IPFS format or Docker registry format",
		Long:  "Convert a hash to a multihash IPFS format or to a format that the Docker registry understands",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrOnlyOneArgumentRequired
			}

			if !(format == "docker" || format == "ipfs") {
				return ErrInvalidConvertFormat
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if format == "docker" {
				ipfsHash := args[0]
				dockerizedHash := regutil.DockerizeHash(ipfsHash)
				fmt.Println(dockerizedHash)
			} else if format == "ipfs" {
				dockerizedHash := args[0]
				ipfsHash := regutil.IpfsifyHash(dockerizedHash)
				fmt.Println(ipfsHash)
			} else {
				return ErrInvalidConvertFormat
			}

			return nil
		},
	}

	convertCmd.Flags().StringVarP(&format, "format", "f", "", "Output format which can be \"docker\" or \"ipfs\"")

	rootCmd.AddCommand(
		pushCmd,
		pullCmd,
		serverCmd,
		convertCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
