package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var namespace string

type Node struct {
	Kind      string
	Name      string
	Status    string
	UID       string
	OwnerRefs []metav1.OwnerReference
	Children  []*Node
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "kubectl-tree <RESOURCE> [NAME]",
		Short: "Display the resource ownership tree for Kubernetes objects",
		Long: `kubectl-tree displays the tree of dependent resources for a given Kubernetes object.
Supported kinds: deployments, statefulsets, daemonsets, jobs, replicasets, pods.

Examples:
  # Show tree for a specific deployment
  kubectl tree deployment nginx

  # Show trees for ALL deployments in the current namespace
  kubectl tree deployment

  # Show tree in a specific namespace
  kubectl tree statefulset redis -n kube-system`,
		Args: cobra.MinimumNArgs(1),
		RunE: runTree,
	}
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTree(cmd *cobra.Command, args []string) error {
	targetKind := strings.ToLower(args[0])
	var targetName string
	if len(args) > 1 {
		targetName = args[1]
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
	roots, err := buildTree(ctx, clientset, namespace, targetKind, targetName)
	if err != nil {
		return err
	}

	if len(roots) == 0 {
		fmt.Printf("No %s found in namespace %q\n", targetKind, namespace)
		return nil
	}

	fmt.Printf("Resource tree in namespace %q\n\n", namespace)
	for i, root := range roots {
		printTree(root, "", i == len(roots)-1)
	}

	return nil
}

func buildTree(ctx context.Context, clientset kubernetes.Interface, ns, targetKind, targetName string) ([]*Node, error) {
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	replicaSets, err := clientset.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list replicasets: %w", err)
	}

	deployments, err := clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	statefulSets, err := clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}

	jobs, err := clientset.BatchV1().Jobs(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	resourceMap := make(map[string]*Node)

	addResource := func(obj metav1.Object, kind string, status string) {
		resourceMap[string(obj.GetUID())] = &Node{
			Kind:      kind,
			Name:      obj.GetName(),
			Status:    status,
			UID:       string(obj.GetUID()),
			OwnerRefs: obj.GetOwnerReferences(),
		}
	}

	for i := range pods.Items {
		p := &pods.Items[i]
		status := string(p.Status.Phase)
		if p.Status.Phase == "Running" && len(p.Status.ContainerStatuses) > 0 {
			ready := 0
			for _, cs := range p.Status.ContainerStatuses {
				if cs.Ready {
					ready++
				}
			}
			status = fmt.Sprintf("%d/%d Ready", ready, len(p.Status.ContainerStatuses))
		}
		addResource(p, "Pod", status)
	}

	for i := range replicaSets.Items {
		rs := &replicaSets.Items[i]
		replicas := int32(1)
		if rs.Spec.Replicas != nil {
			replicas = *rs.Spec.Replicas
		}
		status := fmt.Sprintf("%d/%d Ready", rs.Status.ReadyReplicas, replicas)
		addResource(rs, "ReplicaSet", status)
	}

	for i := range deployments.Items {
		d := &deployments.Items[i]
		replicas := int32(1)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		status := fmt.Sprintf("%d/%d Ready", d.Status.ReadyReplicas, replicas)
		addResource(d, "Deployment", status)
	}

	for i := range statefulSets.Items {
		ss := &statefulSets.Items[i]
		replicas := int32(1)
		if ss.Spec.Replicas != nil {
			replicas = *ss.Spec.Replicas
		}
		status := fmt.Sprintf("%d/%d Ready", ss.Status.ReadyReplicas, replicas)
		addResource(ss, "StatefulSet", status)
	}

	for i := range jobs.Items {
		j := &jobs.Items[i]
		status := "Pending"
		if j.Status.Succeeded > 0 {
			status = "Succeeded"
		} else if j.Status.Failed > 0 {
			status = "Failed"
		} else if j.Status.Active > 0 {
			status = "Running"
		}
		addResource(j, "Job", status)
	}

	for _, node := range resourceMap {
		for _, owner := range node.OwnerRefs {
			if parent, exists := resourceMap[string(owner.UID)]; exists {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	targetKindSingular := strings.TrimSuffix(strings.ToLower(targetKind), "s")
	var roots []*Node
	for _, node := range resourceMap {
		if strings.ToLower(node.Kind) == targetKindSingular {
			if targetName != "" && node.Name != targetName {
				continue
			}
			roots = append(roots, node)
		}
	}

	return roots, nil
}

func printTree(node *Node, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	statusStr := ""
	if node.Status != "" {
		statusStr = " " + formatStatus(node.Status)
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

func formatStatus(status string) string {
	if strings.Contains(status, "Ready") && !strings.HasPrefix(status, "0/") {
		return color.GreenString("(%s)", status)
	}

	if strings.Contains(status, "Failed") || strings.Contains(status, "CrashLoop") || strings.Contains(status, "Evicted") || strings.Contains(status, "Err") {
		return color.RedString("(%s)", status)
	}

	if strings.Contains(status, "Pending") || strings.HasPrefix(status, "0/") {
		return color.YellowString("(%s)", status)
	}

	return color.CyanString("(%s)", status)
}
