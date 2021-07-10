package main

import (
	"bytes"
	"fmt"
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	// dodok8s "github.com/dodopizza/kubectl-shovel/internal/kubernetes" nice sample
	"k8s.io/kubectl/pkg/cmd/cp"
)

func (o *ExecOptions) CopyToPod(from, to string) error {
	to = buildPodPath(o.Namespace, o.PodName, to)

	ioStreams := genericclioptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: os.Stdout,
	}
	opts := cp.NewCopyOptions(ioStreams)
	opts.Clientset = o.ClientSet
	opts.ClientConfig = o.RestConfig

	return opts.Run([]string{from, to})
}

func buildPodPath(namespace, podName, podFilePath string) string {
	return fmt.Sprintf("%s/%s:%s", namespace, podName, podFilePath)
}
