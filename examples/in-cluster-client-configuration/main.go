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
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// get all pods
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("Pods list:")
	for _, p := range pods.Items {
		fmt.Println(p.Name)
	}

	namespace, pod, container, procname, procid := parseEnv()
	if procid != "" || procname != "" {
		// TODO
	}

	// get process list
	err = execCmd(clientset, config, namespace, pod, container, "ps", nil, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Println("remotecommand failed")
		panic(err.Error())
	}

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

func getProcessList(pod, container string) {

}

func execCmd(clientset *kubernetes.Clientset, config *rest.Config, namespace, pod, container, command string,
	stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := []string{
		"sh",
		"-c",
		command,
	}
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod).
		Namespace(namespace).SubResource("exec")
	var option *v1.PodExecOptions
	if container == "" {
		targetpod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Printf("Pod %v not found in default namespace\n", pod)
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("Found pod %v in default namespace\n", pod)
		}
		container = targetpod.Spec.Containers[0].Name
	}
	option = &v1.PodExecOptions{
		Container: container,
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
	fmt.Printf("Getting process list of pod(%v)/container(%v)...\n", pod, container)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	fmt.Println("=====")

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
