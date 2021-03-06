package clustermembercontroller

import (
	"bytes"
	"encoding/json"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	ceoapi "github.com/openshift/cluster-etcd-operator/pkg/operator/api"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

var (
	clusterDomain             = "operator.testing.openshift"
	clusterMembersPendingPath = []string{"cluster", "pending"}
	clusterMembersPath        = []string{"cluster", "members"}
)

func TestClusterMemberController_RemoveBootstrapFromEndpoint(t *testing.T) {

	client := fake.NewSimpleClientset()

	addressList := []v1.EndpointAddress{
		{
			IP:       "192.168.2.1",
			Hostname: "etcd-bootstrap",
		},
		{
			IP:       "192.168.2.2",
			Hostname: "etcd-1",
		},
		{
			IP:       "192.168.2.3",
			Hostname: "etcd-2",
		},
		{
			IP:       "192.168.2.4",
			Hostname: "etcd-3",
		},
	}
	ep := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdHostEndpointName,
			Namespace: EtcdEndpointNamespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: addressList,
			},
		},
	}

	_, err := client.CoreV1().Endpoints(EtcdEndpointNamespace).Create(ep)
	if err != nil {
		t.Fatal()
	}

	type fields struct {
		clientset            kubernetes.Interface
		operatorConfigClient v1helpers.OperatorClient
		queue                workqueue.RateLimitingInterface
		eventRecorder        events.Recorder
		etcdDiscoveryDomain  string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "remove 0th address",
			fields: fields{
				clientset:           client,
				etcdDiscoveryDomain: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterMemberController{
				clientset:            tt.fields.clientset,
				operatorConfigClient: tt.fields.operatorConfigClient,
				queue:                tt.fields.queue,
				eventRecorder:        tt.fields.eventRecorder,
				etcdDiscoveryDomain:  tt.fields.etcdDiscoveryDomain,
			}
			if err := c.RemoveBootstrapFromEndpoint(); (err != nil) != tt.wantErr {
				t.Errorf("RemoveBootstrapFromEndpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getBytes(obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func getEtcdSpec(pending, ready []string) *operatorv1.OperatorSpec {
	observedConfig := map[string]interface{}{}
	etcdPendingMembers := []interface{}{}
	etcdMembers := []interface{}{}

	for _, pm := range pending {
		pendingBucket := map[string]interface{}{}
		if err := unstructured.SetNestedField(pendingBucket, pm+"-node", "name"); err != nil {
			klog.Fatalf("error occured in writing nested fields %#v", err)
		}
		if err := unstructured.SetNestedField(pendingBucket, "https://"+pm+"."+clusterDomain+":2380", "peerURLs"); err != nil {
			klog.Fatalf("error occured in writing nested fields %#v", err)
		}
		if err := unstructured.SetNestedField(pendingBucket, string(ceoapi.MemberUnknown), "status"); err != nil {
			klog.Fatalf("error occured in writing nested fields %#v", err)
		}
		etcdPendingMembers = append(etcdPendingMembers, pendingBucket)
	}
	for _, m := range ready {
		memberBucket := map[string]interface{}{}
		if err := unstructured.SetNestedField(memberBucket, m, "name"); err != nil {
			klog.Fatalf("error occured in writing nested fields %#v", err)
		}
		if err := unstructured.SetNestedField(memberBucket, "https://"+m+"."+clusterDomain+":2380", "peerURLs"); err != nil {
			klog.Fatalf("error occured in writing nested fields %#v", err)
		}
		if err := unstructured.SetNestedField(memberBucket, string(ceoapi.MemberUnknown), "status"); err != nil {
			klog.Fatalf("error occured in writing nested fields %#v", err)
		}
		etcdMembers = append(etcdMembers, memberBucket)
	}
	if len(pending) > 0 {
		if err := unstructured.SetNestedField(observedConfig, etcdPendingMembers, clusterMembersPendingPath...); err != nil {
			klog.Fatalf("error occured in writing pending members: %#v", err)
		}
	}
	if len(ready) > 0 {
		if err := unstructured.SetNestedField(observedConfig, etcdMembers, clusterMembersPath...); err != nil {
			klog.Fatalf("error occured in writing members: %#v", err)
		}
	}
	etcdURLsBytes, err := getBytes(observedConfig)
	if err != nil {
		klog.Fatalf("error occured in getting bytes for etcdURLs: %#v", err)
	}
	return &operatorv1.OperatorSpec{
		ObservedConfig: runtime.RawExtension{
			Raw: etcdURLsBytes,
		},
	}
}

func TestClusterMemberController_isClusterEtcdOperatorReady(t *testing.T) {
	type fields struct {
		operatorConfigClient v1helpers.OperatorClient
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "test with 1 pending member and no ready",
			fields: fields{
				operatorConfigClient: v1helpers.NewFakeOperatorClient(getEtcdSpec([]string{"etcd-1"}, []string{}),
					nil,
					nil),
			},
			want: false,
		},
		{
			name: "test with 0 pending member and no ready members",
			fields: fields{
				operatorConfigClient: v1helpers.NewFakeOperatorClient(getEtcdSpec([]string{}, []string{}),
					nil,
					nil),
			},
			want: false,
		},
		{
			name: "test with 0 pending member and etcd-bootstrap ready",
			fields: fields{
				operatorConfigClient: v1helpers.NewFakeOperatorClient(getEtcdSpec([]string{}, []string{"etcd-bootstrap"}),
					nil,
					nil),
			},
			want: false,
		},
		{
			name: "test with 1 pending member and more than 1 ready",
			fields: fields{
				operatorConfigClient: v1helpers.NewFakeOperatorClient(getEtcdSpec([]string{"etcd-3"}, []string{"etcd-bootstrap", "etcd-1", "etcd-2"}),
					nil,
					nil),
			},
			want: false,
		},
		{
			name: "test with 0 pending member and more than 1 ready",
			fields: fields{
				operatorConfigClient: v1helpers.NewFakeOperatorClient(getEtcdSpec([]string{}, []string{"etcd-bootstrap", "etcd-1", "etcd-2"}),
					nil,
					nil),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterMemberController{
				operatorConfigClient: tt.fields.operatorConfigClient,
			}
			if got := c.isClusterEtcdOperatorReady(); got != tt.want {
				t.Errorf("isClusterEtcdOperatorReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
