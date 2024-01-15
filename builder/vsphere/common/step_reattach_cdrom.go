// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type ReattachCDRomConfig

package common

import (
	"context"
	"fmt"
	"math"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-vsphere/builder/vsphere/driver"
)

type ReattachCDRomConfig struct {
	// Reattach one or more configured CD-ROM devices. Range: 1-4.
	// You can reattach up to 4 CD-ROM devices to the final build artifact.
	// If set to 0, `reattach_cdroms` is ignored and the step is skipped.
	// When set to a value in the range, `remove_cdrom` is ignored and
	// the CD-ROM devices are kept without any attached media.
	ReattachCDRom int `mapstructure:"reattach_cdroms"`
}

type StepReattachCDRom struct {
	Config      *ReattachCDRomConfig
	CDRomConfig *CDRomConfig
}

func (s *StepReattachCDRom) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	vm := state.Get("vm").(driver.VirtualMachine)

	var err error

	// Check if `reattach_cdroms` is set.
	ReattachCDRom := s.Config.ReattachCDRom
	if ReattachCDRom == 0 {
		return multistep.ActionContinue
	}
	if ReattachCDRom < 1 || ReattachCDRom > 4 {
		err := fmt.Errorf("'reattach_cdroms' should be between 1 and 4. if set to 0, `reattach_cdroms` is ignored and the step is skipped")
		state.Put("error", fmt.Errorf("error reattach cdrom: %v", err))
		return multistep.ActionHalt
	}

	ui.Say("Reattaching CD-ROM devices...")
	var n_actable_cdroms int = ReattachCDRom - len(s.CDRomConfig.ISOPaths)
	if n_actable_cdroms < 0 {
		err = vm.RemoveCdroms(int(math.Abs(float64(n_actable_cdroms))))
		if err != nil {
			state.Put("error", fmt.Errorf("error removing cdrom prior to reattaching: %v", err))
			return multistep.ActionHalt
		}
		ui.Say("Ejecting CD-ROM media...")
		// Eject media from CD-ROM devices.
		err = vm.EjectCdroms()
		if err != nil {
			state.Put("error", fmt.Errorf("error ejecting cdrom media: %v", err))
			return multistep.ActionHalt
		}
	} else {
		// Eject media from pre-existing CD-ROM devices.
		err = vm.EjectCdroms()
		if err != nil {
			state.Put("error", fmt.Errorf("error ejecting cdrom media: %v", err))
			return multistep.ActionHalt
		}

		err = vm.RemoveCdroms(0)
		if err != nil {
			state.Put("error", fmt.Errorf("error removing cdrom prior to reattaching: %v", err))
			return multistep.ActionHalt
		}
		// create more CD-ROMs if required

		// If the CD-ROM device type is SATA, make sure SATA controller is present.
		if s.CDRomConfig.CdromType == "sata" {
			if _, err := vm.FindSATAController(); err == driver.ErrNoSataController {
				ui.Say("Adding SATA controller...")
				if err := vm.AddSATAController(); err != nil {
					state.Put("error", fmt.Errorf("error adding sata controller: %v", err))
					return multistep.ActionHalt
				}
			}
		}

		if n_actable_cdroms > 0 {
			ui.Say("Adding CD-ROM devices...")
			_, err := vm.MakeCdroms(s.CDRomConfig.CdromType, n_actable_cdroms, true)
			if err != nil {
				state.Put("error", err)
				return multistep.ActionHalt
			}
		}
	}
	return multistep.ActionContinue
}

func (s *StepReattachCDRom) Cleanup(state multistep.StateBag) {
	// no cleanup
}
