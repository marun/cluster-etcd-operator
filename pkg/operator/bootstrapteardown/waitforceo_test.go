package bootstrapteardown

import (
	"testing"

	v12 "k8s.io/api/core/v1"

	operatorv1 "github.com/openshift/api/operator/v1"
	v1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_doneEtcd(t *testing.T) {
	type args struct {
		etcd *operatorv1.Etcd
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "test unmanaged cluster",
			args: args{
				etcd: &v1.Etcd{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: v1.EtcdSpec{
						StaticPodOperatorSpec: v1.StaticPodOperatorSpec{
							OperatorSpec: v1.OperatorSpec{
								ManagementState: v1.Unmanaged,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test managed cluster but degraded",
			args: args{
				etcd: &v1.Etcd{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: v1.EtcdSpec{
						StaticPodOperatorSpec: v1.StaticPodOperatorSpec{
							OperatorSpec: v1.OperatorSpec{
								ManagementState: v1.Managed,
							},
						},
					},
					Status: v1.EtcdStatus{
						StaticPodOperatorStatus: v1.StaticPodOperatorStatus{
							OperatorStatus: v1.OperatorStatus{
								Conditions: []v1.OperatorCondition{
									{
										Type:   v1.OperatorStatusTypeDegraded,
										Status: v1.ConditionTrue,
									},
								},
							},
						}},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "test managed cluster but progressing",
			args: args{
				etcd: &v1.Etcd{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: v1.EtcdSpec{
						StaticPodOperatorSpec: v1.StaticPodOperatorSpec{
							OperatorSpec: v1.OperatorSpec{
								ManagementState: v1.Managed,
							},
						},
					},
					Status: v1.EtcdStatus{
						StaticPodOperatorStatus: v1.StaticPodOperatorStatus{
							OperatorStatus: v1.OperatorStatus{
								Conditions: []v1.OperatorCondition{
									{
										Type:   v1.OperatorStatusTypeProgressing,
										Status: v1.ConditionTrue,
									},
								},
							},
						}},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "test managed cluster but unavailable",
			args: args{
				etcd: &v1.Etcd{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: v1.EtcdSpec{
						StaticPodOperatorSpec: v1.StaticPodOperatorSpec{
							OperatorSpec: v1.OperatorSpec{
								ManagementState: v1.Managed,
							},
						},
					},
					Status: v1.EtcdStatus{
						StaticPodOperatorStatus: v1.StaticPodOperatorStatus{
							OperatorStatus: v1.OperatorStatus{
								Conditions: []v1.OperatorCondition{
									{
										Type:   v1.OperatorStatusTypeAvailable,
										Status: v1.ConditionFalse,
									},
								},
							},
						}},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "test managed cluster and doneEtcd",
			args: args{
				etcd: &v1.Etcd{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: v1.EtcdSpec{
						StaticPodOperatorSpec: v1.StaticPodOperatorSpec{
							OperatorSpec: v1.OperatorSpec{
								ManagementState: v1.Managed,
							},
						},
					},
					Status: v1.EtcdStatus{
						StaticPodOperatorStatus: v1.StaticPodOperatorStatus{
							OperatorStatus: v1.OperatorStatus{
								Conditions: []v1.OperatorCondition{
									{
										Type:   v1.OperatorStatusTypeDegraded,
										Status: v1.ConditionFalse,
									},
									{
										Type:   v1.OperatorStatusTypeProgressing,
										Status: v1.ConditionFalse,
									},
									{
										Type:   v1.OperatorStatusTypeAvailable,
										Status: v1.ConditionTrue,
									},
								},
							},
						}},
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := doneEtcd(tt.args.etcd)
			if (err != nil) != tt.wantErr {
				t.Errorf("doneEtcd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("doneEtcd() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_doneApiServer(t *testing.T) {
	// TODO: implement me
}

func Test_configMapHasRequiredValues(t *testing.T) {
	type args struct {
		configMap *v12.ConfigMap
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "three urls",
			args: args{
				configMap: &v12.ConfigMap{Data: map[string]string{
					configMapKey: `{
							"storageConfig": {
								"urls":[
									"https://etcd-1.foo.bar:2379",
									"https://etcd-0.foo.bar:2379",
									"https://etcd-2.foo.bar:2379"
								]
							}
						}`,
				},
				},
			},
			want: true,
		},
		{
			name: "2 urls but not bootstrap",
			args: args{
				configMap: &v12.ConfigMap{Data: map[string]string{
					configMapKey: `{
							"storageConfig": {
								"urls":[
									"https://etcd-1.foo.bar:2379",
									"https://etcd-2.foo.bar:2379"
								]
							}
						}`,
				},
				},
			},
			want: true,
		},
		{
			name: "single urls but not bootstrap",
			args: args{
				configMap: &v12.ConfigMap{Data: map[string]string{
					configMapKey: `{
							"storageConfig": {
								"urls":[
									"https://etcd-1.foo.bar:2379"
								]
							}
						}`,
				},
				},
			},
			want: true,
		},
		{
			name: "just the bootstrap url",
			args: args{
				configMap: &v12.ConfigMap{Data: map[string]string{
					configMapKey: `{
							"storageConfig": {
								"urls": [
									"https://10.13.14.15:2379"
								]
							}
						}`,
				},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configMapHasRequiredValues(tt.args.configMap); got != tt.want {
				t.Errorf("configMapHasRequiredValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
