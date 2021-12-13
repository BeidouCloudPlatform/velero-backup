package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ergoapi/util/ptr"
	"github.com/ergoapi/util/ztime"
	"github.com/spf13/cobra"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/builder"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/flag"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	veleroclient "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const DefaultBackupTTL time.Duration = 30 * 24 * time.Hour

type BackupOptions struct {
	Name                    string
	TTL                     time.Duration
	SnapshotVolumes         flag.OptionalBool
	DefaultVolumesToRestic  flag.OptionalBool
	IncludeNamespaces       flag.StringArray
	ExcludeNamespaces       flag.StringArray
	IncludeResources        flag.StringArray
	ExcludeResources        flag.StringArray
	Labels                  flag.Map
	Selector                flag.LabelSelector
	IncludeClusterResources flag.OptionalBool
	StorageLocation         string
	SnapshotLocations       []string
	FromSchedule            string
	OrderedResources        string

	client veleroclient.Interface
}

// newBackupCmd ergo k8s backup
func newBackupCmd() *cobra.Command {
	opt := &BackupOptions{
		Name:              ztime.NowUnixString(),
		TTL:               DefaultBackupTTL,
		IncludeNamespaces: flag.NewStringArray("*"),
		Labels:            flag.NewMap(),
		Selector: flag.LabelSelector{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.easycorp.work/managed-by": "cce",
					"k8s.easycorp.work/appid":      "e2a0718e-3311-41a2-ae4d-9be03c51af1d",
					"k8s.easycorp.work/name":       "go",
				},
			},
		},
		SnapshotVolumes:         flag.NewOptionalBool(ptr.BoolPtr(true)),
		IncludeClusterResources: flag.NewOptionalBool(ptr.BoolPtr(true)),
	}
	b := &cobra.Command{
		Use:     "backup [flags]",
		Short:   "backup",
		Version: "2.0.7",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			cmd.CheckError(opt.Complete(args))
			cmd.CheckError(opt.Validate(cobraCmd, args))
			cmd.CheckError(opt.Run(cobraCmd))
			return nil
		},
	}
	return b
}

func (o *BackupOptions) Complete(args []string) error {
	dir, _ := os.UserHomeDir()
	kubecfg := filepath.Join(dir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubecfg)
	if err != nil {
		return err
	}
	client, err := veleroclient.NewForConfig(config)
	if err != nil {
		return err
	}
	o.client = client
	return nil
}

func (o *BackupOptions) Validate(c *cobra.Command, args []string) error {
	for _, loc := range o.SnapshotLocations {
		if _, err := o.client.VeleroV1().VolumeSnapshotLocations("velero").Get(context.TODO(), loc, metav1.GetOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (o *BackupOptions) Run(c *cobra.Command) error {
	backup, err := o.BuildBackup("velero")
	if err != nil {
		return err
	}

	if printed, err := output.PrintWithFormat(c, backup); printed || err != nil {
		return err
	}

	_, err = o.client.VeleroV1().Backups(backup.Namespace).Create(context.TODO(), backup, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("Backup request %q submitted successfully.\n", backup.Name)
	// Not waiting

	fmt.Printf("Run `velero backup describe %s` or `velero backup logs %s` for more details.\n", backup.Name, backup.Name)

	return nil
}

func (o *BackupOptions) BuildBackup(namespace string) (*velerov1api.Backup, error) {
	var backupBuilder *builder.BackupBuilder

	backupBuilder = builder.ForBackup(namespace, o.Name).
		IncludedNamespaces(o.IncludeNamespaces...).
		ExcludedNamespaces(o.ExcludeNamespaces...).
		IncludedResources(o.IncludeResources...).
		ExcludedResources(o.ExcludeResources...).
		LabelSelector(o.Selector.LabelSelector).
		TTL(o.TTL).
		StorageLocation(o.StorageLocation).
		VolumeSnapshotLocations(o.SnapshotLocations...)
	if o.SnapshotVolumes.Value != nil {
		backupBuilder.SnapshotVolumes(*o.SnapshotVolumes.Value)
	}
	if o.IncludeClusterResources.Value != nil {
		backupBuilder.IncludeClusterResources(*o.IncludeClusterResources.Value)
	}
	if o.DefaultVolumesToRestic.Value != nil {
		backupBuilder.DefaultVolumesToRestic(*o.DefaultVolumesToRestic.Value)
	}

	backup := backupBuilder.ObjectMeta(builder.WithLabelsMap(o.Labels.Data())).Result()
	return backup, nil
}
