package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// HeadscaleObject is implemented by both Headscale and ClusterHeadscale,
// allowing controllers to handle both resource types uniformly.
// +kubebuilder:object:generate=false
type HeadscaleObject interface {
	metav1.Object
	runtime.Object
	GetHeadscaleSpec() *HeadscaleSpec
	GetHeadscaleStatus() *HeadscaleStatus
	GetTargetNamespace() string
}

func (h *Headscale) GetHeadscaleSpec() *HeadscaleSpec {
	return &h.Spec
}

func (h *Headscale) GetHeadscaleStatus() *HeadscaleStatus {
	return &h.Status
}

func (h *Headscale) GetTargetNamespace() string {
	return h.Namespace
}

func (c *ClusterHeadscale) GetHeadscaleSpec() *HeadscaleSpec {
	return &c.Spec.HeadscaleSpec
}

func (c *ClusterHeadscale) GetHeadscaleStatus() *HeadscaleStatus {
	return &c.Status
}

func (c *ClusterHeadscale) GetTargetNamespace() string {
	return c.Spec.TargetNamespace
}
