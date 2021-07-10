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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

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
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"} // this is required when using kubectl/cp, don't know why not in exec
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

	opts.parseEnv()

	// dump steps:
	// 0. validate pod, detect os (sh or powershell)
	// 1. output process list, `ps` or `Get-Process`
	// 2. create dump, only support windows for now

	// Step 0
	err = opts.validatePod()
	checkErr(err, "")

	// Step 1
	opts.getProcessList()

	// Step 2
	if opts.PodOS == "windows" {
		if opts.ProcID == "0" && opts.ProcName == "" {
			fmt.Println("Please re-create a processdump resource and specify the process-id")
			// block
			<-(chan int)(nil)
		} else {
			opts.watsonDump()
		}
	} else {
		// TODO, linux dump
		fmt.Println("Dump in linux container not supported yet.")
	}

	fmt.Println("WorkerPod finished.")

	// block
	<-(chan int)(nil)
}

func (o *ExecOptions) parseEnv() {
	o.Namespace = os.Getenv("NAMESPACE")
	o.PodName = os.Getenv("POD_NAME")
	if o.PodName == "" {
		panic("pod name cannot be empty")
	}
	o.ContainerName = os.Getenv("CONTAINER_NAME")
	o.ProcName = os.Getenv("PROCESS_NAME")
	o.ProcID = os.Getenv("PROCESS_ID")
}

func (o *ExecOptions) validatePod() error {
	// get pod
	targetpod, err := o.ClientSet.CoreV1().Pods(o.Namespace).Get(context.TODO(), o.PodName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Pod %v not found in Namespace %v\n", o.PodName, o.Namespace)
		// block
		<-(chan int)(nil)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting Pod %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found pod %v in Namespace %v\n", o.PodName, o.Namespace)
	}
	if o.ContainerName == "" {
		o.ContainerName = targetpod.Spec.Containers[0].Name
	} else {
		// TODO, validate given container exists in pod
	}
	// fmt.Printf("%+v\n\n", targetpod.Spec)

	// get OS
	var found bool
	if o.PodOS, found = targetpod.Spec.NodeSelector["beta.kubernetes.io/os"]; !found {
		if o.PodOS, found = targetpod.Spec.NodeSelector["kubernetes.io/os"]; !found {
			fmt.Printf("found no nodeSelector in targetpod(%v), assume is linux\n", o.PodName)
			o.PodOS = "linux"
			return nil
		}
	}
	fmt.Println("Pod OS: " + o.PodOS)
	return nil
}

func (o *ExecOptions) getProcessList() (output string, err error) {
	fmt.Printf("Getting process list of pod(%v)/container(%v)...\n\n", o.PodName, o.ContainerName)

	if o.PodOS == "linux" {
		cmd := []string{
			"sh",
			"-c",
			"ps",
		}
		err = o.execCmd(cmd, nil, os.Stdout, os.Stderr)
	} else {
		var b bytes.Buffer
		cmd := []string{
			"powershell.exe",
			"Get-Process",
		}
		err = o.execCmd(cmd, nil, &b, os.Stderr)
		fmt.Println(b.String())
		fmt.Println()
	}
	checkErr(err, "remotecommand failed")
	fmt.Println("")

	return output, err
}

func (o *ExecOptions) watsonDump() error {
	err := o.CopyToPod("/run-dump.ps1", "run-dump.ps1")
	checkErr(err, "")
	// err = o.CopyToPod("/start.cosmic.ps1", "JitWatson/start.cosmic.ps1")
	// checkErr(err, "")

	scriptparam := "C:\\run-dump.ps1 "
	if o.ProcID != "0" {
		scriptparam += "-ProcID " + o.ProcID
	} else {
		scriptparam += "-ProcName " + o.ProcName
	}
	cmd := []string{
		"powershell.exe",
		scriptparam,
	}

	fmt.Println("Start dump ... ")
	var weird bytes.Buffer
	err = o.execCmd(cmd, nil, &weird, nil)
	checkErr(err, "")
	// fmt.Println()
	// fmt.Println()
	// fmt.Println()
	// fmt.Println()
	// fmt.Println()
	// fmt.Println(weird.String())
	// fmt.Println()
	// fmt.Println()
	// fmt.Println()
	// fmt.Println()
	// fmt.Println()
	// time.Sleep(20 * time.Second)
	var b bytes.Buffer
	for i := 0; i < 20; i++ {
		cmd := []string{
			"powershell.exe",
			"cat log.txt",
		}
		o.execCmd(cmd, nil, &b, nil)
		if b.Len() > 0 {
			// fmt.Println("============= Dump Uploaded ==============")
			fmt.Println(b.String())
			return nil
		} else {
			time.Sleep(2 * time.Second)
		}
	}
	panic("Dump process failed.")
}

func (o *ExecOptions) execCmd(cmd []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	req := o.ClientSet.CoreV1().RESTClient().Post().Resource("pods").Name(o.PodName).
		Namespace(o.Namespace).SubResource("exec")

	option := &v1.PodExecOptions{
		Container: o.ContainerName,
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
	// fmt.Printf("Executing cmd: %+v", cmd)
	// fmt.Println()

	// errorChan := remotecommand.watchErrorStream(stderr, &errorDecoderV2{})
	exec, err := remotecommand.NewSPDYExecutor(o.RestConfig, "POST", req.URL())

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
	time.Sleep(1000 * time.Millisecond)
	fmt.Println()

	return nil
	// return <-errorChan
}

type ExecOptions struct {
	ClientSet     kubernetes.Interface
	RestConfig    *rest.Config
	PodName       string
	ContainerName string
	Namespace     string
	PodOS         string
	ProcName      string
	ProcID        string
}

func checkErr(err error, msg string) {
	if err != nil {
		if msg != "" {
			fmt.Println(msg)
		}
		panic(err.Error())
	}
}
