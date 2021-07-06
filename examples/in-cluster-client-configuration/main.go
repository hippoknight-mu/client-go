/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func main() {

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	checkErr(err, "")

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err, "")

	k8sExec := &K8sExec{
		ClientSet:  clientset,
		RestConfig: config,
	}

	// get all pods
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	checkErr(err, "")

	fmt.Println("Pods list:")
	for _, p := range pods.Items {
		fmt.Println(p.Name)
	}
	fmt.Printf("total pod number: %v\n\n", len(pods.Items))

	namespace, pod, container, procname, procid := parseEnv()
	if procid != "" || procname != "" {
		// TODO
	}

	k8sExec.PodName = pod
	k8sExec.ContainerName = container
	k8sExec.Namespace = namespace

	// get process list
	k8sExec.getProcessList()

	// copy dump tool binary
	k8sExec.TestCopyToPod()
	k8sExec.execCmd("ls -a /tmp", nil, os.Stdout, os.Stderr)

	// block
	<-(chan int)(nil)
}

func parseArg() (pod, container string) {
	argsWithoutProg := os.Args[1:]
	fmt.Println(len(argsWithoutProg))
	for idx, arg := range argsWithoutProg {
		fmt.Printf("arg-%v: %v\n", idx, arg)
	}
	return
}

func parseEnv() (namespace, pod, container, procname, procid string) {
	namespace = os.Getenv("NAMESPACE")
	pod = os.Getenv("POD_NAME")
	container = os.Getenv("CONTAINER_NAME")
	if pod == "" {
		panic("pod name cannot be empty")
	}
	procname = os.Getenv("PROC_NAME")
	procid = os.Getenv("PROC_ID")
	return
}

func (k *K8sExec) getProcessList() {
	fmt.Printf("Getting process list of pod(%v)/container(%v)...\n", k.PodName, k.ContainerName)
	// if os linux
	err := k.execCmd("ps", nil, os.Stdout, os.Stderr)
	checkErr(err, "remotecommand failed")
	fmt.Println("=====")
}

func (k *K8sExec) execCmd(command string,
	stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := []string{
		"sh",
		"-c",
		command,
	}
	req := k.ClientSet.CoreV1().RESTClient().Post().Resource("pods").Name(k.PodName).
		Namespace(k.Namespace).SubResource("exec")
	var option *v1.PodExecOptions
	if k.ContainerName == "" {
		targetpod, err := k.ClientSet.CoreV1().Pods(k.Namespace).Get(context.TODO(), k.PodName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Printf("Pod %v not found in default namespace\n", k.PodName)
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("Found pod %v in default namespace\n", k.PodName)
		}
		k.ContainerName = targetpod.Spec.Containers[0].Name
	}
	option = &v1.PodExecOptions{
		Container: k.ContainerName,
		Command:   cmd,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	if stdin == nil {
		option.Stdin = false
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(k.RestConfig, "POST", req.URL())
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    true,
	})
	if err != nil {
		return err
	}

	return nil
}

// func cpFile(filename string) {
// 	dir, err := ioutil.TempDir("", "input")
// 	checkErr(err, "")
// 	filepath := path.Join(dir, filename)

// 	opt := &cp.CopyOptions{
// 		IOStreams: genericclioptions.NewTestIOStreamsDiscard(),
// 	}

// 	src := cp.fileSpec{
// 		File: src,
// 	}
// 	dest := cp.fileSpec{
// 		PodNamespace: "pod-ns",
// 		PodName:      "pod-name",
// 		File:         dest,
// 	}
// 	err = opt.copyToPod(src, dest, &kexec.ExecOptions{})

// 	// writer := &bytes.Buffer{}
// 	// if err := makeTar(dir, dir, writer); err != nil {
// 	// 	t.Fatalf("unexpected error: %v", err)
// 	// }

// 	// reader := bytes.NewBuffer(writer.Bytes())
// 	// if err := opts.untarAll(fileSpec{}, reader, dir2, ""); err != nil {
// 	// 	t.Fatalf("unexpected error: %v", err)
// 	// }
// }

type K8sExec struct { // https://zhimin-wen.medium.com/programing-exec-into-a-pod-5f2a70bd93bb
	ClientSet     kubernetes.Interface
	RestConfig    *rest.Config
	PodName       string
	ContainerName string
	Namespace     string
}

// func (k8s *K8sExec) Exec(command []string) ([]byte, []byte, error) {
// 	req := k8s.ClientSet.CoreV1().RESTClient().Post().
// 		Resource("pods").
// 		Name(k8s.PodName).
// 		Namespace(k8s.Namespace).
// 		SubResource("exec")
// 	req.VersionedParams(&v1.PodExecOptions{
// 		Container: k8s.ContainerName,
// 		Command:   command,
// 		Stdin:     false,
// 		Stdout:    true,
// 		Stderr:    true,
// 		TTY:       true,
// 	}, scheme.ParameterCodec)
// 	log.Infof("Request URL: %s", req.URL().String())
// 	exec, err := remotecommand.NewSPDYExecutor(k8s.RestConfig, "POST", req.URL())
// 	if err != nil {
// 		log.Errorf("Failed to exec:%v", err)
// 		return []byte{}, []byte{}, err
// 	}
// 	var stdout, stderr bytes.Buffer
// 	err = exec.Stream(remotecommand.StreamOptions{
// 		Stdin:  nil,
// 		Stdout: &stdout,
// 		Stderr: &stderr,
// 	})
// 	if err != nil {
// 		log.Errorf("Faile to get result:%v", err)
// 		return []byte{}, []byte{}, err
// 	}
// 	return stdout.Bytes(), stderr.Bytes(), nil
// }

func checkErr(err error, msg string) {
	if err != nil {
		if msg != "" {
			fmt.Println(msg)
		}
		panic(err.Error())
	}
}
