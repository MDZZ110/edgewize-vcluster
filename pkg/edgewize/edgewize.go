package edgewize

import (
	"context"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	fakenodes = &sync.Map{}
	once = sync.Once{}
)


func IsSystemWorkspace(cli client.Client, name string) (bool, error) {
	namespace := &corev1.Namespace{}
	err := cli.Get(context.Background(), types.NamespacedName{Name: name}, namespace)
	if err != nil {
		return false, err
	}
	return namespace.Labels["kubesphere.io/workspace"] == "system-workspace", nil
}

func IsFakeNode(cli client.Client, name string) (bool, error) {
	if _, ok := fakenodes.Load(name); ok {
		return true, nil
	}
	node := &corev1.Node{}
	err := cli.Get(context.Background(), types.NamespacedName{Name: name}, node)
	if err != nil {
		klog.Errorf("failed to get node %s: %v", name, err)
		return false, err
	} else {
		if node.Labels["vcluster.loft.sh/fake-node"] == "true" {
			klog.Errorf("node is not fake node, but has label %s", name)
		}
	}
	return false, nil
}

func AddFakeNode(name string) {
	fakenodes.Store(name, struct{}{})
}

func InitFakeNode(ctx *synccontext.RegisterContext) {
	once.Do(func() {
		podList := &corev1.PodList{}
		err := ctx.PhysicalManager.GetClient().List(ctx.Context, podList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"vcluster.loft.sh/managed-by": translate.Suffix,
			}),
		})
		if err != nil {
			klog.Errorf("error listing pods: %v", err)
		} else {
			klog.Infof("edgewize.FakeNodes found %d pods", len(podList.Items))
			for _, pod := range podList.Items {
				if pod.Spec.NodeName != "" {
					AddFakeNode(pod.Spec.NodeName)
					klog.Infof("edgewize.FakeNodes added fake node %s", pod.Spec.NodeName)
				}
			}
		}
	})
}