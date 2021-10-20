package main

import (
	"context"
	"fmt"
	"log"
	"os"

	operatorv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	OperatorV1Alpha1SchemeGV = schema.GroupVersion{Group: "operators.coreos.com", Version: "v1alpha1"}
)

func main() {

	// Approach 1: dynamic client
	scheme := runtime.NewScheme()
	operatorv1alpha1.AddToScheme(scheme)

	kubeconfig := ctrl.GetConfigOrDie()
	controllerClient, err := runtimeclient.New(kubeconfig, runtimeclient.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)
		return
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
	var catalogSourceKind schema.GroupVersionKind = schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Kind:    "CatalogSource",
		Version: "v1alpha1",
	}
	u.SetGroupVersionKind(catalogSourceKind)

	err = controllerClient.Create(context.Background(), u)
	if err != nil {
		log.Fatal("unable to create resource: ", err)
		return
	}

	err = controllerClient.Get(context.Background(), runtimeclient.ObjectKey{
		Namespace: "default",
		Name:      "cs-dynamic",
	}, u)

	if err != nil {
		log.Fatal("unable to retrieve resource: ", err)
		return
	}

	fmt.Println(convert(u))

	//Approach 2: Typed clients

	var k8sconfig *rest.Config

	if len(os.Getenv("KUBECONFIG")) > 0 {
		k8sconfig, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))

		if err != nil {
			log.Fatal("unable to parse kubeconfig: ", err)
			return
		}
	}

	// create a custom scheme by adding Go types
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		log.Fatal(err)
		return
	}
	k8sconfig.GroupVersion = &OperatorV1Alpha1SchemeGV
	k8sconfig.APIPath = "/apis"
	k8sconfig.ContentType = runtime.ContentTypeJSON
	k8sconfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme)

	// Approach 2.1: create a rest client
	client, err := rest.RESTClientFor(k8sconfig)
	if err != nil {
		log.Fatal(err)
		return
	}

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

	result := &operatorv1alpha1.CatalogSource{}
	err = client.Post().
		Namespace("default").
		Resource("catalogsources").
		Body(obj).
		Do(context.Background()).
		Into(result)

	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println(result)

	// Approach 2.2: custom Go client via controller-runtime
	// the client is able to handle any kind that is registered in a given scheme

	cl, err := runtimeclient.New(k8sconfig, runtimeclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	obj1 := &operatorv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cs-typed-client",
			Namespace: "default",
		},
		Spec: operatorv1alpha1.CatalogSourceSpec{
			SourceType:  operatorv1alpha1.SourceTypeGrpc,
			Image:       "sbarouti/gitlab-runner-operator-indeximage:v0.0.1-04925b2b",
			DisplayName: "CS - Typed Client",
		},
	}

	err = cl.Create(context.Background(), obj1)
	if err != nil {
		log.Fatal(err)
		return
	}

}

func addKnownTypes(scheme *runtime.Scheme) error {

	scheme.AddKnownTypes(OperatorV1Alpha1SchemeGV,
		&operatorv1alpha1.CatalogSource{},
		&operatorv1alpha1.CatalogSourceList{},
	)
	metav1.AddToGroupVersion(scheme, OperatorV1Alpha1SchemeGV)

	return nil
}

func convert(u *unstructured.Unstructured) (*operatorv1alpha1.CatalogSource, error) {
	var obj operatorv1alpha1.CatalogSource
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(u.UnstructuredContent(), &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}
