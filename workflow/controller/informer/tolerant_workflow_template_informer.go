package informer

import (
	"fmt"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	extwfv1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
)

type tolerantWorkflowTemplateInformer struct {
	delegate informers.GenericInformer
}

// a drop-in replacement for `extwfv1.WorkflowTemplateInformer` that ignores malformed resources
func NewTolerantWorkflowTemplateInformer(dynamicInterface dynamic.Interface, defaultResync time.Duration, namespace string) extwfv1.WorkflowTemplateInformer {
	informer := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicInterface, defaultResync, namespace, func(options *metav1.ListOptions) {
		// `ResourceVersion=0` does not honor the `limit` in API calls, which results in making significant List calls
		// without `limit`. For details, see https://github.com/argoproj/argo-workflows/pull/11343
		options.ResourceVersion = ""
	}).
		ForResource(schema.GroupVersionResource{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowTemplatePlural})
	informer.Informer().SetTransform(transform)
	return &tolerantWorkflowTemplateInformer{delegate: informer}
}

func transform(obj interface{}) (interface{}, error) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return obj, fmt.Errorf("malformed workflow template: expected \"*unstructured.Unstructured\", got \"%s\"", reflect.TypeOf(obj).String())
	}
	tmpl, err := interfaceToWorkflowTemplate(obj)
	if err != nil {
		return obj, fmt.Errorf("malformed workflow template \"%s/%s\": %w", un.GetNamespace(), un.GetName(), err)
	}

	un.Object["workflowTemplate"] = tmpl
	delete(un.Object, "status")
	return un, nil
}

func (t *tolerantWorkflowTemplateInformer) Informer() cache.SharedIndexInformer {
	return t.delegate.Informer()
}

func (t *tolerantWorkflowTemplateInformer) Lister() v1alpha1.WorkflowTemplateLister {
	return &tolerantWorkflowTemplateLister{delegate: t.delegate.Lister()}
}
