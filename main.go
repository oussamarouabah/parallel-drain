package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/drain"
)

var (
	concurrency   int
	nodeSelector  string
	oldVersion    string
	checkInterval time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "parallel-drain",
	Short: "Continuously drains multiple nodes of a specific version in parallel",
	RunE:  run,
}

func init() {
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of nodes to drain in parallel")
	rootCmd.Flags().StringVarP(&nodeSelector, "selector", "l", "", "Label selector to filter nodes")
	rootCmd.Flags().StringVar(&oldVersion, "old-k8s-version", "", "The Kubernetes version to match and drain (e.g. v1.32.0)")
	rootCmd.Flags().DurationVar(&checkInterval, "interval", 10*time.Second, "Interval to check for nodes to drain")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if oldVersion == "" {
		return fmt.Errorf("--old-k8s-version is required")
	}

	fmt.Printf("Starting parallel drain loop for version %s with concurrency %d, checking every %s...\n", oldVersion, concurrency, checkInterval)

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	var (
		nodeChan        = make(chan string)
		processingNodes sync.Map
		wg              sync.WaitGroup

		drainHelper = &drain.Helper{
			Client:              clientset,
			Force:               false,
			IgnoreAllDaemonSets: true,
			DeleteEmptyDirData:  true,
			GracePeriodSeconds:  30,
			Out:                 os.Stdout,
			ErrOut:              os.Stderr,
		}
	)

	for i := 0; i < concurrency; i++ {
		wg.Go(func() {
			for nodeName := range nodeChan {
				fmt.Printf("Processing node: %s\n", nodeName)

				node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting node %s: %v\n", nodeName, err)
					processingNodes.Delete(nodeName)
					continue
				}

				if err := drain.RunCordonOrUncordon(drainHelper, node, true); err != nil {
					fmt.Fprintf(os.Stderr, "Error cordoning %s: %v\n", nodeName, err)
					processingNodes.Delete(nodeName)
					continue
				}
				if err := drain.RunNodeDrain(drainHelper, nodeName); err != nil {
					fmt.Fprintf(os.Stderr, "Error draining %s: %v\n", nodeName, err)
				} else {
					fmt.Printf("Successfully drained node: %s\n", nodeName)
				}

				processingNodes.Delete(nodeName)
			}
		})
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
			LabelSelector: nodeSelector,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing nodes: %v\n", err)
			continue
		}

		var oldNode int
		for _, node := range nodes.Items {
			if node.Status.NodeInfo.KubeletVersion != oldVersion {
				continue
			}
			oldNode++
			if _, loading := processingNodes.LoadOrStore(node.Name, true); loading {
				continue
			}

			nodeChan <- node.Name
		}

		if oldNode == 0 {
			break
		}
	}

	close(nodeChan)
	wg.Wait()

	fmt.Println("All matching nodes have been drained.")
	return nil
}
