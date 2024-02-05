package main

import (
	"context"
	"flag"
	"fmt"
	"os/exec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Kubernetes config 파일 경로 지정
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "absolute path to the kubeconfig file")
	flag.Parse()

	// kubeconfig 파일을 사용하여 Kubernetes 클라이언트 설정 로드
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Kubernetes 클라이언트 생성
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// 모든 노드 가져오기
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// 각 노드에서 systemctl status scini 실행
	for _, node := range nodes.Items {
		nodeName := node.ObjectMeta.Name
		cmd := exec.Command("kubectl", "exec", nodeName, "--", "systemctl", "status", "scini")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error getting status for node %s: %v\n", nodeName, err)
			continue
		}
		fmt.Printf("Status for node %s:\n%s\n", nodeName, string(out))
	}
}

