package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespace string
)

type Node struct {
	Kind     string
	Name     string
	Status   string
	Children []*Node
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "kubectl-tree RESOURCE [NAME]",
		Short: "Display the resource tree for a Kubernetes object",
		Long: `kubectl-tree displays the tree of dependent resources for a given Kubernetes object.

Examples:
  # Show tree for a specific deployment
  kubectl tree deployment nginx

  # Show trees for all deployments in namespace
  kubectl tree deployment

  # Show in specific namespace
  kubectl tree deployment nginx -n kube-system`,
		Args: cobra.MinimumNArgs(1),
		RunE: runTree,
	}

	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTree(cmd *cobra.Command, args []string) error {
	var resourceName string
	if len(args) > 1 {
		resourceName = args[1]
	}

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

	ctx := context.Background()

	tree := buildTree(ctx, clientset, namespace, resourceName)

	fmt.Printf("Resource tree in namespace %q\n\n", namespace)
	if tree != nil {
		printTree(tree, "", true)
	} else {
		fmt.Println("No resources found")
	}

	return nil
}

func buildTree(ctx context.Context, clientset kubernetes.Interface, namespace, deploymentName string) *Node {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	allResources := make(map[string]*Node)

	for _, rs := range replicaSets.Items {
		status := fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, *rs.Spec.Replicas)
		allResources[string(rs.UID)] = &Node{
			Kind:   "ReplicaSet",
			Name:   rs.Name,
			Status: status,
		}
	}

	for _, pod := range pods.Items {
		status := string(pod.Status.Phase)
		if pod.Status.Phase == "Running" && len(pod.Status.ContainerStatuses) > 0 {
			ready := 0
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Ready {
					ready++
				}
			}
			status = fmt.Sprintf("Running (%d/%d)", ready, len(pod.Status.ContainerStatuses))
		}
		allResources[string(pod.UID)] = &Node{
			Kind:   "Pod",
			Name:   pod.Name,
			Status: status,
		}
	}

	for _, rs := range replicaSets.Items {
		for _, owner := range rs.OwnerReferences {
			if parent, exists := allResources[string(owner.UID)]; exists {
				parent.Children = append(parent.Children, allResources[string(rs.UID)])
			}
		}
	}

	for _, pod := range pods.Items {
		for _, owner := range pod.OwnerReferences {
			if parent, exists := allResources[string(owner.UID)]; exists {
				parent.Children = append(parent.Children, allResources[string(pod.UID)])
			}
		}
	}

	for _, dep := range deployments.Items {
		if deploymentName != "" && dep.Name != deploymentName {
			continue
		}

		status := fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, *dep.Spec.Replicas)
		rootNode := &Node{
			Kind:   "Deployment",
			Name:   dep.Name,
			Status: status,
		}

		for _, rs := range replicaSets.Items {
			for _, owner := range rs.OwnerReferences {
				if owner.Kind == "Deployment" && owner.Name == dep.Name {
					rootNode.Children = append(rootNode.Children, allResources[string(rs.UID)])
				}
			}
		}

		return rootNode
	}

	return nil
}

func printTree(node *Node, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	statusStr := ""
	if node.Status != "" {
		statusStr = fmt.Sprintf(" [%s]", node.Status)
	}
	fmt.Printf("%s%s%s/%s%s\n", prefix, connector, node.Kind, node.Name, statusStr)

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	for i, child := range node.Children {
		isLastChild := (i == len(node.Children)-1)
		printTree(child, childPrefix, isLastChild)
	}
}
