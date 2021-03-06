/*
Copyright 2017 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package restore

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/heptio/ark/pkg/client"
	"github.com/heptio/ark/pkg/cmd"
	"github.com/heptio/ark/pkg/cmd/util/flag"
	"github.com/heptio/ark/pkg/cmd/util/output"
)

func NewCreateCommand(f client.Factory) *cobra.Command {
	o := NewCreateOptions()

	c := &cobra.Command{
		Use:   "create BACKUP",
		Short: "Create a restore",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Validate(c, args))
			cmd.CheckError(o.Complete(args))
			cmd.CheckError(o.Run(c, f))
		},
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type CreateOptions struct {
	BackupName        string
	RestoreVolumes    bool
	Labels            flag.Map
	Namespaces        flag.StringArray
	NamespaceMappings flag.Map
	Selector          flag.LabelSelector
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Labels:            flag.NewMap(),
		NamespaceMappings: flag.NewMap().WithEntryDelimiter(",").WithKeyValueDelimiter(":"),
	}
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.RestoreVolumes, "restore-volumes", o.RestoreVolumes, "whether to restore volumes from snapshots")
	flags.Var(&o.Labels, "labels", "labels to apply to the restore")
	flags.Var(&o.Namespaces, "namespaces", "comma-separated list of namespaces to restore")
	flags.Var(&o.NamespaceMappings, "namespace-mappings", "namespace mappings from name in the backup to desired restored name in the form src1:dst1,src2:dst2,...")
	flags.VarP(&o.Selector, "selector", "l", "only restore resources matching this label selector")
}

func (o *CreateOptions) Validate(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("you must specify only one argument, the backup's name")
	}

	if err := output.ValidateFlags(c); err != nil {
		return err
	}

	return nil
}

func (o *CreateOptions) Complete(args []string) error {
	o.BackupName = args[0]
	return nil
}

func (o *CreateOptions) Run(c *cobra.Command, f client.Factory) error {
	arkClient, err := f.Client()
	if err != nil {
		return err
	}

	restore := &api.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: api.DefaultNamespace,
			Name:      fmt.Sprintf("%s-%s", o.BackupName, time.Now().Format("20060102150405")),
			Labels:    o.Labels.Data(),
		},
		Spec: api.RestoreSpec{
			BackupName:       o.BackupName,
			Namespaces:       o.Namespaces,
			NamespaceMapping: o.NamespaceMappings.Data(),
			LabelSelector:    o.Selector.LabelSelector,
			RestorePVs:       o.RestoreVolumes,
		},
	}

	if printed, err := output.PrintWithFormat(c, restore); printed || err != nil {
		return err
	}

	restore, err = arkClient.ArkV1().Restores(restore.Namespace).Create(restore)
	if err != nil {
		return err
	}

	fmt.Printf("Restore %q created successfully.\n", restore.Name)
	return nil
}
