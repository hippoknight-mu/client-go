package main

import (
	// "bytes"
	// "io/ioutil"
	// "net/http"

	// v1 "k8s.io/api/core/v1"
	// "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	// "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubectl/pkg/cmd/cp"
)

func (k8sExec *K8sExec) TestCopyToPod() {
	// tf := cmdtesting.NewTestFactory().WithNamespace("test")
	// f :=
	// ns := scheme.Codecs.WithoutConversion()
	// codec := scheme.Codecs.LegacyCodec(scheme.Scheme.PrioritizedVersionsAllGroups()...)

	// tf.ClientConfigVal = cmdtesting.DefaultClientConfig()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams()
	// ioStreams := genericclioptions.N

	// srcFile, err := ioutil.TempDir("", "test")
	// if err != nil {
	// 	t.Errorf("unexpected error: %v", err)
	// 	t.FailNow()
	// }
	// // defer os.RemoveAll(srcFile)

	opts := cp.NewCopyOptions(ioStreams)
	opts.ClientConfig = k8sExec.RestConfig
	opts.Clientset = k8sExec.ClientSet
	opts.Container = k8sExec.ContainerName
	opts.Namespace = k8sExec.Namespace

	// cmd := cp.NewCmdCp(tf, ioStreams)
	// opts.Complete(tf, cmd)

	src, dest := "/app", k8sExec.PodName+":/tmp/app"
	opts.Run([]string{src, dest})
}
