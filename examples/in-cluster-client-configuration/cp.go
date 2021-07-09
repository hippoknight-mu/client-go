package main

import (
	"bytes"
	"fmt"
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	// dodok8s "github.com/dodopizza/kubectl-shovel/internal/kubernetes" nice sample
	"k8s.io/kubectl/pkg/cmd/cp"
)

func (k *ExecOptions) TestCopyToPod() {
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams()

	opts := cp.NewCopyOptions(ioStreams)
	opts.ClientConfig = k.RestConfig
	opts.Clientset = k.ClientSet
	opts.Container = k.ContainerName
	opts.Namespace = k.Namespace

	src, dest := "/app", k.PodName+":/tmp/app"
	err := opts.Run([]string{src, dest})
	checkErr(err, "cp failed")
}

func (k *ExecOptions) TestDoDoCopy(from, to string) error {
	to = buildPodPath(k.Namespace, k.PodName, to)

	ioStreams := genericclioptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: os.Stdout,
	}
	opts := cp.NewCopyOptions(ioStreams)
	opts.Clientset = k.ClientSet
	opts.ClientConfig = k.RestConfig

	return opts.Run([]string{from, to})
}

func buildPodPath(namespace, podName, podFilePath string) string {
	return fmt.Sprintf("%s/%s:%s", namespace, podName, podFilePath)
}
