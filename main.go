package main

import (
	"context"
	"log"
	"os"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	OperatorV1Alpha1SchemeGV = schema.GroupVersion{Group: "operators.coreos.com", Version: "v1alpha1"}
)

func main() {
	dynamicClient()
	restClient()
	customGoClient()
}

func dynamicClient() error {

	k8sconfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.Fatal("unable to parse kubeconfig: ", err)
		return err
	}

	controllerClient, err := runtimeclient.New(k8sconfig, runtimeclient.Options{})
	if err != nil {
		log.Fatal(err)
		return err
	}
	u := &unstructured.Unstructured{}
	u.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "cs-dynamic",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"sourceType":  operatorv1alpha1.SourceTypeGrpc,
			"image":       "sbarouti/gitlab-runner-operator-indeximage:v0.0.1-04925b2b",
			"displayName": "CS - Dynamic Client",
		},
	}

	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Kind:    "CatalogSource",
		Version: "v1alpha1",
	})

	err = controllerClient.Create(context.Background(), u)
	if err != nil {
		log.Fatal("unable to create resource: ", err)
		return err
	}
	return nil
}

// Approach 2.1: create a rest client
func restClient() error {

	k8sconfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))

	if err != nil {
		log.Fatal("unable to parse kubeconfig: ", err)
		return err
	}

	scheme := runtime.NewScheme()

	// create a custom scheme by adding Go types
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		log.Fatal(err)
		return err
	}
	k8sconfig.GroupVersion = &OperatorV1Alpha1SchemeGV
	k8sconfig.APIPath = "/apis"
	k8sconfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme)

	obj := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cs-rest-client",
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType:  operatorv1alpha1.SourceTypeGrpc,
			Image:       "sbarouti/gitlab-runner-operator-indeximage:v0.0.1-04925b2b",
			DisplayName: "CS - Rest Client",
		},
	}

	client, err := rest.RESTClientFor(k8sconfig)
	if err != nil {
		log.Fatal(err)
		return err
	}

	result := &operatorv1alpha1.CatalogSource{}
	err = client.Post().
		Namespace("default").
		Resource("catalogsources").
		Body(obj).
		Do(context.Background()).
		Into(result)

	return err
}

// Approach 2.2: custom Go client via controller-runtime
// the client is able to handle any kind that is registered in a given scheme
func customGoClient() error {
	k8sconfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))

	if err != nil {
		log.Fatal("unable to parse kubeconfig: ", err)
		return err
	}
	// create a custom scheme by adding Go types
	scheme := runtime.NewScheme()

	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		log.Fatal(err)
		return err
	}

	cl, err := runtimeclient.New(k8sconfig, runtimeclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Fatal(err)
		return err
	}

	obj := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cs-go-client",
			Namespace: "default",
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType:  operatorv1alpha1.SourceTypeGrpc,
			Image:       "sbarouti/gitlab-runner-operator-indeximage:v0.0.1-04925b2b",
			DisplayName: "CS - Custom Go Client",
		},
	}

	err = cl.Create(context.Background(), obj)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(OperatorV1Alpha1SchemeGV,
		&operatorv1alpha1.CatalogSource{},
		&operatorv1alpha1.CatalogSourceList{},
	)
	metav1.AddToGroupVersion(scheme, OperatorV1Alpha1SchemeGV)

	return nil
}
