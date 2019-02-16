package main

import (
	"errors"
	"fmt"
	"os"

	color "github.com/fatih/color"
	registry "github.com/miguelmota/ipdr/registry"
	cobra "github.com/spf13/cobra"
)

var green = color.New(color.FgGreen)

var (
	// ErrImageIDRequired is error for when image ID is required
	ErrImageIDRequired = errors.New("image hash or name is required")
	// ErrOnlyOneArgumentRequired is error for when one argument only is required
	ErrOnlyOneArgumentRequired = errors.New("only one argument is required")
)

func main() {
	var ipfsHost string

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
				DockerLocalRegistryHost: "5000",
				IPFSHost:                ipfsHost,
			})

			imageID := args[0]

			hash, err := reg.PushImageByID(imageID)
			if err != nil {
				return err
			}

			fmt.Println(green.Sprintf("\nSuccessfully pushed Docker image to IPFS:\n/ipfs/%s", hash))
			return nil
		},
	}

	pushCmd.Flags().StringVarP(&ipfsHost, "ipfs-host", "", "127.0.0.1:5001", "A remote IPFS API host to push the image to. Example: 127.0.0.1:5001")

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
				DockerLocalRegistryHost: "docker.localhost",
			})

			imageHash := args[0]
			_, err := reg.PullImage(imageHash)

			return err
		},
	}

	rootCmd.AddCommand(
		pushCmd,
		pullCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
