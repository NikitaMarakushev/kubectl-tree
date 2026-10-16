package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespace string
	allNs     bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kubectl-tree RESOURCE NAME",
		Short: "Display the resource tree for a Kubernetes object",
		Long: `kubectl-tree displays the tree of dependent resources for a given Kubernetes object.

Examples:
  # Show tree for a deployment
  kubectl tree deployment nginx

  # Show tree in specific namespace
  kubectl tree deployment nginx -n kube-system

  # Show trees for all deployments
  kubectl tree deployment --all`,
		Args: cobra.MinimumNArgs(1),
		RunE: runTree,
	}

	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	rootCmd.Flags().BoolVarP(&allNs, "all-namespaces", "A", false, "Show trees in all namespaces")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTree(cmd *cobra.Command, args []string) error {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	resourceType := args[0]
	var resourceName string
	if len(args) > 1 {
		resourceName = args[1]
	}

	fmt.Printf("Building tree for %s", resourceType)
	if resourceName != "" {
		fmt.Printf("/%s", resourceName)
	}
	fmt.Printf(" in namespace %q\n\n", namespace)

	_ = clientset
	return nil
}
