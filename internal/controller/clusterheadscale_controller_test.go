package controller

import (
	"context"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	headscalev1beta2 "github.com/infradohq/headscale-operator/api/v1beta2"
)

var _ = Describe("ClusterHeadscale Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName       = "test-clusterheadscale"
			readyConditionType = "Ready"
			timeout            = time.Second * 10
			interval           = time.Millisecond * 250
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name: resourceName,
		}

		AfterEach(func() {
			By("Cleaning up the test ClusterHeadscale")
			for _, name := range []string{resourceName, "cross-ns-clusterheadscale"} {
				ch := &headscalev1beta2.ClusterHeadscale{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, ch)
				if err == nil {
					_ = k8sClient.Delete(ctx, ch)
				}
			}

			By("Waiting for the base StatefulSet to be cleaned up")
		})

		It("should successfully create and reconcile a ClusterHeadscale instance", func() {
			By("Creating the ClusterHeadscale resource")
			ch := &headscalev1beta2.ClusterHeadscale{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
				Spec: headscalev1beta2.ClusterHeadscaleSpec{
					TargetNamespace: "default",
					HeadscaleSpec: headscalev1beta2.HeadscaleSpec{
						Version:  "v0.28.0",
						Replicas: 1,
						Config: headscalev1beta2.HeadscaleConfig{
							ServerURL:      "https://clusterheadscale.example.com",
							ListenAddr:     "0.0.0.0:8080",
							GRPCListenAddr: "0.0.0.0:50443",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, ch)).To(Succeed())

			By("Checking that the ClusterHeadscale was created")
			createdCH := &headscalev1beta2.ClusterHeadscale{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdCH)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Reconciling the created resource")
			controllerReconciler := &HeadscaleReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the finalizer was added")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdCH)
				if err != nil {
					return false
				}
				return slices.Contains(createdCH.Finalizers, headscaleFinalizer)
			}, timeout, interval).Should(BeTrue())

			By("Verifying status condition was set")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdCH)
				if err != nil {
					return false
				}
				for _, condition := range createdCH.Status.Conditions {
					if condition.Type == readyConditionType &&
						condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should allow resources in different namespaces to reference it", func() {
			const crossNSResourceName = "cross-ns-clusterheadscale"
			otherNamespace := "other-namespace"
			By("Creating the other namespace")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: otherNamespace,
				},
			}
			// Use k8sClient to create namespace if it doesn't exist
			_ = k8sClient.Create(ctx, ns)

			By("Creating the ClusterHeadscale resource with a custom namespace")
			ch := &headscalev1beta2.ClusterHeadscale{
				ObjectMeta: metav1.ObjectMeta{
					Name: crossNSResourceName,
				},
				Spec: headscalev1beta2.ClusterHeadscaleSpec{
					TargetNamespace: otherNamespace,
					HeadscaleSpec: headscalev1beta2.HeadscaleSpec{
						Version:  "v0.28.0",
						Replicas: 1,
						Config: headscalev1beta2.HeadscaleConfig{
							ServerURL:      "https://clusterheadscale.example.com",
							ListenAddr:     "0.0.0.0:8080",
							GRPCListenAddr: "0.0.0.0:50443",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, ch)).To(Succeed())

			By("Creating a HeadscalePreAuthKey in a different namespace")
			keyName := "test-key"
			key := &headscalev1beta2.HeadscalePreAuthKey{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyName,
					Namespace: otherNamespace,
				},
				Spec: headscalev1beta2.HeadscalePreAuthKeySpec{
					HeadscaleRef: headscalev1beta2.HeadscaleRef{
						Name: crossNSResourceName,
						Kind: "ClusterHeadscale",
					},
					Tags: []string{"tag:test"},
				},
			}
			Expect(k8sClient.Create(ctx, key)).To(Succeed())

			By("Reconciling the HeadscalePreAuthKey")
			keyReconciler := &HeadscalePreAuthKeyReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := keyReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      keyName,
					Namespace: otherNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the PreAuthKey found the ClusterHeadscale")
			updatedKey := &headscalev1beta2.HeadscalePreAuthKey{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: keyName, Namespace: otherNamespace}, updatedKey)
				if err != nil {
					return false
				}
				// It shouldn't have "HeadscaleNotFound" condition
				for _, cond := range updatedKey.Status.Conditions {
					if cond.Type == "Ready" && cond.Reason == "HeadscaleNotFound" {
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("should reject a ClusterHeadscale without spec.targetNamespace", func() {
			ch := &headscalev1beta2.ClusterHeadscale{
				ObjectMeta: metav1.ObjectMeta{
					Name: "missing-target-namespace",
				},
				Spec: headscalev1beta2.ClusterHeadscaleSpec{
					HeadscaleSpec: headscalev1beta2.HeadscaleSpec{
						Version:  "v0.28.0",
						Replicas: 1,
						Config: headscalev1beta2.HeadscaleConfig{
							ServerURL:  "https://example.com",
							ListenAddr: "0.0.0.0:8080",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, ch)).NotTo(Succeed())
		})
	})
})
