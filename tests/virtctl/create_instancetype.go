package virtctl

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"
	"kubevirt.io/client-go/kubecli"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/util"
)

const namespaced = "--namespaced"

var _ = Describe("[sig-compute] create instancetype", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
	var clusterInstancetype *instancetypev1alpha2.VirtualMachineClusterInstancetype

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if clusterInstancetype != nil {
			Expect(virtClient.VirtualMachineClusterInstancetype().Delete(context.Background(), clusterInstancetype.Name, metav1.DeleteOptions{})).To(Succeed())
			clusterInstancetype = nil
		}
	})

	createInstancetypeSpec := func(bytes []byte, namespaced bool) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error) {
		decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		switch obj := decodedObj.(type) {
		case *instancetypev1alpha2.VirtualMachineInstancetype:
			ExpectWithOffset(1, namespaced).To(BeTrue(), "expected VirtualMachineInstancetype to be created")
			ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineInstancetype"))
			clusterInstancetype, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), (*instancetypev1alpha2.VirtualMachineClusterInstancetype)(obj), metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return &clusterInstancetype.Spec, nil
		case *instancetypev1alpha2.VirtualMachineClusterInstancetype:
			ExpectWithOffset(1, namespaced).To(BeFalse(), "expected VirtualMachineClusterInstancetype to be created")
			ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineClusterInstancetype"))
			instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).Create(context.Background(), (*instancetypev1alpha2.VirtualMachineInstancetype)(obj), metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return &instancetype.Spec, nil
		default:
			return nil, fmt.Errorf("object must be VirtualMachineInstance or VirtualMachineClusterInstancetype")
		}
	}

	Context("should create valid instancetype manifest", func() {
		DescribeTable("when CPU and Memory defined", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.CPU.Guest).To(Equal(uint32(2)))
			Expect(instancetypeSpec.Memory.Guest).To(Equal(resource.MustParse("256Mi")))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("when GPUs defined", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(GPUFlag, "name:gpu1,devicename:nvidia/gpu1"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.GPUs[0].Name).To(Equal("gpu1"))
			Expect(instancetypeSpec.GPUs[0].DeviceName).To(Equal("nvidia/gpu1"))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("when hostDevice defined", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(HostDeviceFlag, "name:device1,devicename:hostdevice1"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.HostDevices[0].Name).To(Equal("device1"))
			Expect(instancetypeSpec.HostDevices[0].DeviceName).To(Equal("hostdevice1"))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("when IOThreadsPolicy defined", func(namespacedFlag, policyStr string, namespaced bool, policy v1.IOThreadsPolicy) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(IOThreadsPolicyFlag, policyStr),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := createInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(*instancetypeSpec.IOThreadsPolicy).To(Equal(policy))
		},
			Entry("VirtualMachineInstancetype", namespaced, "auto", true, v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineClusterInstancetype", "", "shared", false, v1.IOThreadsPolicyShared),
		)
	})
})
