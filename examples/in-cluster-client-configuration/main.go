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
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"} // this is required using kubectl/cp, don't know why not in exec
	if config.APIPath == "" {
		config.APIPath = "/api"
	}
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}
	if len(config.UserAgent) == 0 {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err, "")

	opts := &ExecOptions{
		ClientSet:  clientset,
		RestConfig: config,
	}

	namespace, pod, container, procname, procid := parseEnv()

	opts.ContainerName = container
	opts.PodName = pod
	opts.Namespace = namespace

	if procid != "" || procname != "" {
		// TODO
	}

	// dump steps:
	// 0. validate pod, detect os (sh or powershell)
	// 1. output process list, `ps` or `Get-Process`
	// 2. create dump, only support windows for now

	// Step 0
	err = opts.validatePod()
	checkErr(err, "")

	// Step 1
	err = opts.getProcessList()
	checkErr(err, "")

	// Step 2
	if opts.PodOS == "windows" {
		err = opts.watsonDump(procid)
		checkErr(err, "")
	}

	// block
	<-(chan int)(nil)

	// // get all pods
	// pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	// checkErr(err, "")

	// fmt.Println("Pods list:")
	// for _, p := range pods.Items {
	// 	fmt.Println(p.Name)
	// }
	// fmt.Printf("total pod number: %v\n\n", len(pods.Items))

	// get process list
	opts.getProcessList()

	// copy dump tool binary
	opts.TestCopyToPod()
	// err = ExecOptions.TestDoDoCopy("/app", "/tmp/app")
	checkErr(err, "cp failed")
	// opts.execCmd(["ls -a /tmp"], nil, os.Stdout, os.Stderr)

}

// func parseArg() (pod, container string) {
// 	argsWithoutProg := os.Args[1:]
// 	fmt.Println(len(argsWithoutProg))
// 	for idx, arg := range argsWithoutProg {
// 		fmt.Printf("arg-%v: %v\n", idx, arg)
// 	}
// 	return
// }

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

func (k *ExecOptions) validatePod() error {
	targetpod, err := k.ClientSet.CoreV1().Pods(k.Namespace).Get(context.TODO(), k.PodName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Pod %v not found in Namespace %v\n", k.PodName, k.Namespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting Pod %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found pod %v in Namespace %v\n", k.PodName, k.Namespace)
	}
	if k.ContainerName == "" {
		k.ContainerName = targetpod.Spec.Containers[0].Name
	} else {
		// TODO, validate given container exists in pod
	}

	fmt.Printf("%+v\n\n", targetpod.Spec)

	var found bool
	// get OS
	if k.PodOS, found = targetpod.Spec.NodeSelector["beta.kubernetes.io/os"]; !found {
		if k.PodOS, found = targetpod.Spec.NodeSelector["kubernetes.io/os"]; !found {
			fmt.Printf("found no nodeSelector in targetpod(%v), assume is linux\n", k.PodName)
			k.PodOS = "linux"
			return nil
		}
	}
	fmt.Println("Pod OS is " + k.PodOS)
	return nil
}

func (k *ExecOptions) getProcessList() (err error) {
	fmt.Printf("Getting process list of pod(%v)/container(%v)...\n", k.PodName, k.ContainerName)
	if k.PodOS == "linux" {
		cmd := []string{
			"sh",
			"-c",
			"ps",
		}
		err = k.execCmd(cmd, nil, os.Stdout, os.Stderr)
	} else {
		cmd := []string{
			"powershell.exe",
			"Get-Process",
		}
		err = k.execCmd(cmd, nil, os.Stdout, os.Stderr)
	}
	checkErr(err, "remotecommand failed")
	fmt.Println("=====")
	return err
}

func (k *ExecOptions) watsonDump(procid string) error {
	cmd := []string{
		"powershell.exe",
		"C:\\JitWatson\\start.cosmic.ps1",
	}
	err := k.execCmd(cmd, nil, os.Stdout, os.Stderr)
	checkErr(err, "")

	cmd = []string{
		"powershell.exe",
		"C:\\JitWatson\\Dump-CrashReportingProcess.ps1 -UniquePid 8524",
	}
	err = k.execCmd(cmd, nil, os.Stdout, os.Stderr)
	checkErr(err, "")
	return nil
}

func (k *ExecOptions) execCmd(cmd []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	// var option *v1.PodExecOptions

	req := k.ClientSet.CoreV1().RESTClient().Post().Resource("pods").Name(k.PodName).
		Namespace(k.Namespace).SubResource("exec")

	option := &v1.PodExecOptions{
		Container: k.ContainerName,
		Command:   cmd,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
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

type ExecOptions struct { // https://zhimin-wen.medium.com/programing-exec-into-a-pod-5f2a70bd93bb
	ClientSet     kubernetes.Interface
	RestConfig    *rest.Config
	PodName       string
	ContainerName string
	Namespace     string
	PodOS         string
}

// func (k8s *ExecOptions) Exec(command []string) ([]byte, []byte, error) {
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
