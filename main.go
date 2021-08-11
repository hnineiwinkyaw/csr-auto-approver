package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	certs "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	certificate "k8s.io/api/certificates/v1beta1"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "log"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func approveCsrs(csr certs.CertificateSigningRequestInterface, pod core.PodInterface) bool {
	return true
}

type Message struct {
    Type string
    Reason string
}

func main() {

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Initialise configuration
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	panic(err.Error())
	// }
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
	// Set up the csr client
		certificateRestClient := clientset.CertificatesV1beta1().RESTClient()
		certificateClient := certs.New(certificateRestClient)
		csr := certificateClient.CertificateSigningRequests()

		csrs, err := csr.List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		
		fmt.Printf("There are %d csr in the cluster\n", len(csrs.Items))

		for i, csrObj := range csrs.Items {
			fmt.Println(i)
		    fmt.Println(csrObj.Name)
		    fmt.Printf("%+v\n",csrObj.Status.Conditions)

		    if len(csrObj.Status.Conditions) > 0 && csrObj.Status.Conditions[0].Type == certificate.CertificateApproved {
		    	fmt.Printf("Already approved")
		    } else {
		    	csrObj.Status.Conditions = append(csrObj.Status.Conditions, certificate.CertificateSigningRequestCondition{
					Type:			certificate.CertificateApproved,
					Reason: 		"handle by csr auto approver",
					Message:		"This CSR was approved by csr-approver-app",
					LastUpdateTime: metav1.Now(),
				})

			    fmt.Printf("%+v\n", csrObj.Status.Conditions)

			    clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(context.TODO(),&csrObj,metav1.UpdateOptions{})
		    }
		    
		    fmt.Printf("\n\n\n\n")
		}
		
		// FIXME: only working in the default namespace
		// pod := clientset.CoreV1().Pods("default")

		// log.Fatalln(approveCsrs(csr, pod))

		time.Sleep(50 * time.Second)
	}
}