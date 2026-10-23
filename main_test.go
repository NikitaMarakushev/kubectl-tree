package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildTree(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
			UID:       "dep-uid-123",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
		},
	}

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-abc123",
			Namespace: "default",
			UID:       "rs-uid-456",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "nginx",
					UID:  "dep-uid-123",
				},
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: int32Ptr(3),
		},
		Status: appsv1.ReplicaSetStatus{
			ReadyReplicas: 3,
		},
	}

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-abc123-xyz1",
			Namespace: "default",
			UID:       "pod-uid-789",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "ReplicaSet",
					Name: "nginx-abc123",
					UID:  "rs-uid-456",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
			ContainerStatuses: []corev1.ContainerStatus{
				{Ready: true},
			},
		},
	}

	objects := []runtime.Object{deployment, replicaSet, pod1}
	clientset := fake.NewSimpleClientset(objects...)

	ctx := context.Background()

	trees, err := buildTree(ctx, clientset, "default", "deployment", "nginx")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(trees) == 0 {
		t.Fatal("Expected at least one tree to be built, got empty slice")
	}

	tree := trees[0]

	if tree.Kind != "Deployment" {
		t.Errorf("Expected root Kind to be Deployment, got %s", tree.Kind)
	}

	if tree.Name != "nginx" {
		t.Errorf("Expected root Name to be nginx, got %s", tree.Name)
	}

	if len(tree.Children) != 1 {
		t.Fatalf("Expected 1 ReplicaSet child, got %d", len(tree.Children))
	}

	rs := tree.Children[0]
	if rs.Kind != "ReplicaSet" {
		t.Errorf("Expected child Kind to be ReplicaSet, got %s", rs.Kind)
	}

	if len(rs.Children) != 1 {
		t.Fatalf("Expected 1 Pod child, got %d", len(rs.Children))
	}

	pod := rs.Children[0]
	if pod.Kind != "Pod" {
		t.Errorf("Expected grandchild Kind to be Pod, got %s", pod.Kind)
	}

	if !strings.Contains(pod.Status, "Ready") {
		t.Errorf("Expected Pod status to contain 'Ready', got %s", pod.Status)
	}
}

func TestBuildTreeMultipleRoots(t *testing.T) {
	dep1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "dep1", Namespace: "default", UID: "uid1"},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(1)},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
	}
	dep2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "dep2", Namespace: "default", UID: "uid2"},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(1)},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 0},
	}

	objects := []runtime.Object{dep1, dep2}
	clientset := fake.NewSimpleClientset(objects...)

	ctx := context.Background()
	trees, err := buildTree(ctx, clientset, "default", "deployment", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(trees) != 2 {
		t.Fatalf("Expected 2 trees to be returned, got %d", len(trees))
	}
}

func TestPrintTree(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tree := &Node{
		Kind:   "Deployment",
		Name:   "nginx",
		Status: "3/3 Ready",
		Children: []*Node{
			{
				Kind:   "ReplicaSet",
				Name:   "nginx-abc",
				Status: "3/3 Ready",
				Children: []*Node{
					{
						Kind:   "Pod",
						Name:   "nginx-abc-xyz",
						Status: "Running (1/1)",
					},
				},
			},
		},
	}

	printTree(tree, "", true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Deployment/nginx") {
		t.Errorf("Expected output to contain 'Deployment/nginx', got: %s", output)
	}

	if !strings.Contains(output, "ReplicaSet/nginx-abc") {
		t.Errorf("Expected output to contain 'ReplicaSet/nginx-abc', got: %s", output)
	}

	if !strings.Contains(output, "Pod/nginx-abc-xyz") {
		t.Errorf("Expected output to contain 'Pod/nginx-abc-xyz', got: %s", output)
	}

	if !strings.Contains(output, "└──") {
		t.Errorf("Expected output to contain tree connector '└──', got: %s", output)
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
