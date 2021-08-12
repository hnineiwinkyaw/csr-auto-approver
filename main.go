package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"time"
	"strings"

	"k8s.io/client-go/kubernetes"
	certs "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	certificate "k8s.io/api/certificates/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"encoding/pem"
	"crypto/x509"
)

func getCertificateCommonName(csrObj *certificate.CertificateSigningRequest) string {
	block, _ := pem.Decode(csrObj.Spec.Request)

	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		fmt.Printf(block.Type)
		fmt.Printf("PEM block type must be CERTIFICATE REQUEST")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		fmt.Printf("Error in parsing cr")
	}

	// fmt.Printf("%+v\n",csr.IPAddresses)

	return csr.Subject.CommonName
}

func extractCSRStatus(csr *certificate.CertificateSigningRequest) (string, error) {
	var approved, denied bool
	for _, c := range csr.Status.Conditions {
		switch c.Type {
		case certificate.CertificateApproved:
			approved = true
		case certificate.CertificateDenied:
			denied = true
		default:
			return "", fmt.Errorf("unknown csr condition %q", c)
		}
	}
	var status string
	// must be in order of presidence
	if denied {
		status += "Denied"
	} else if approved {
		status += "Approved"
	} else {
		status += "Pending"
	}
	if len(csr.Status.Certificate) > 0 {
		status += ",Issued"
	}
	return status, nil
}

func doesCnResolvesIpAddr(commonName string, pods *core.PodList) bool {

	if  !strings.HasSuffix(commonName, ".pod.cluster.local") {
		fmt.Println("No Suffix with pod.cluster.local")
		return false
	}

	podIpPlusNs := strings.Split(commonName,".pod.cluster.local")[0]

	fmt.Printf("Pod IP and namespace : %s \n", podIpPlusNs)

	for _, pod := range pods.Items {
		
		fmt.Println("Filtering by namespace\n")

		if  strings.HasSuffix(podIpPlusNs, pod.Namespace) {
			fmt.Printf("Pod found with ns : %s, getting pod ip ... \n", pod.Namespace)
			podIp := strings.Split(podIpPlusNs,"."+pod.Namespace)
			fmt.Printf("Pod IP : %s \n", podIp[0])
			fmt.Printf("Filtering actual pod by certificate ca podIp ... \n")
			if podIp[0] == pod.Status.PodIP {
				fmt.Printf("Found Pod : %s in namespace %s" , pod.Name, pod.Namespace)
				return true;
			}
		}
	}
	return false
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

		pods, err := clientset.CoreV1().Pods("").List(context.TODO(),metav1.ListOptions{})
		if err != nil {
			fmt.Println("Error in listing pods")
		}


		for _, csrObj := range csrs.Items {

			status, err := extractCSRStatus(&csrObj)

			if err != nil {
				panic(err.Error())
			}

			fmt.Printf("Certificate status : %s \n", status)

		    if strings.HasPrefix(status, "Pending") {

		    	commonName := getCertificateCommonName(&csrObj)

				fmt.Printf("Common Name is %s \n", commonName)

			    validPod := doesCnResolvesIpAddr(commonName, pods)			

			    if validPod {
			    	fmt.Printf("Approving Certificate ... \n")
			    	csrObj.Status.Conditions = append(csrObj.Status.Conditions, certificate.CertificateSigningRequestCondition{
						Type:			certificate.CertificateApproved,
						Reason: 		"handled by csr auto approver",
						Message:		"This CSR was approved by csr-approver-app",
						LastUpdateTime: metav1.Now(),
					})

					clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(context.TODO(),&csrObj,metav1.UpdateOptions{})
			    }
		    	
		    }
		}
		
		fmt.Printf("Sleeping for 10 second ... \n\n")

		time.Sleep(10 * time.Second)
	}
}