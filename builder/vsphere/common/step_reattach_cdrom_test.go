// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-vsphere/builder/vsphere/driver"
)

func TestStepReattachCDRom_Run(t *testing.T) {
	tc := []struct {
		name           string
		step           *StepReattachCDRom
		expectedAction multistep.StepAction
		vmMock         *driver.VirtualMachineMock
		expectedVmMock *driver.VirtualMachineMock
		fail           bool
		errMessage     string
	}{
		{
			name: "Successfully attach 3 additional CD-ROM devices",
			step: &StepReattachCDRom{
				Config: &ReattachCDRomConfig{
					ReattachCDRom: 4,
				},
				CDRomConfig: &CDRomConfig{
					CdromType: "sata",
					ISOPaths:  []string{"[datastore] /iso/linux.iso"},
				},
			},
			expectedAction: multistep.ActionContinue,
			vmMock: &driver.VirtualMachineMock{
				ReattachCDRomsCalled: true,
			},
			expectedVmMock: &driver.VirtualMachineMock{
				EjectCdromsCalled:        true,
				ReattachCDRomsCalled:     true,
				FindSATAControllerCalled: true,
				AddCdromCalledTimes:      6,
				AddCdromTypes: []string{
					"sata", "sata",
					"sata", "sata",
					"sata", "sata",
				},
			},
			fail: false,
		},
		{
			name: "Successfully attach 2 fewer CD-ROM devices than initially added",
			step: &StepReattachCDRom{
				Config: &ReattachCDRomConfig{
					ReattachCDRom: 2,
				},
				CDRomConfig: &CDRomConfig{
					CdromType: "sata",
					ISOPaths: []string{
						"[datastore] /iso/linux1.iso",
						"[datastore] /iso/linux2.iso",
						"[datastore] /iso/linux3.iso",
						"[datastore] /iso/linux4.iso",
					},
				},
			},
			expectedAction: multistep.ActionContinue,
			vmMock: &driver.VirtualMachineMock{
				ReattachCDRomsCalled: true,
			},
			expectedVmMock: &driver.VirtualMachineMock{
				RemoveNCdromsCalled:  true,
				EjectCdromsCalled:    true,
				ReattachCDRomsCalled: true,
				AddCdromCalledTimes:  0,
			},
			fail: false,
		},
		{
			name: "Successfully attach the same number of CD-ROMs as initially added",
			step: &StepReattachCDRom{
				Config: &ReattachCDRomConfig{
					ReattachCDRom: 2,
				},
				CDRomConfig: &CDRomConfig{
					CdromType: "sata",
					ISOPaths: []string{
						"[datastore] /iso/linux1.iso",
						"[datastore] /iso/linux2.iso",
					},
				},
			},
			expectedAction: multistep.ActionContinue,
			vmMock: &driver.VirtualMachineMock{
				ReattachCDRomsCalled: true,
			},
			expectedVmMock: &driver.VirtualMachineMock{
				EjectCdromsCalled:    true,
				ReattachCDRomsCalled: true,
				AddCdromCalledTimes:  0,
			},
			fail: false,
		},
	}
	tc = append(tc, struct {
		name           string
		step           *StepReattachCDRom
		expectedAction multistep.StepAction
		vmMock         *driver.VirtualMachineMock
		expectedVmMock *driver.VirtualMachineMock
		fail           bool
		errMessage     string
	}{
		name: "Fail to reattach CD-ROM device",
		step: &StepReattachCDRom{
			Config: &ReattachCDRomConfig{
				ReattachCDRom: 5,
			},
			CDRomConfig: &CDRomConfig{
				CdromType: "sata",
				ISOPaths:  []string{"[datastore] /iso/linux.iso"},
			},
		},
		expectedAction: multistep.ActionHalt,
		vmMock:         &driver.VirtualMachineMock{},
		expectedVmMock: &driver.VirtualMachineMock{
			ReattachCDRomsErr: fmt.Errorf("'reattach_cdroms' should be between 1 and 4"),
		},
		fail:       true,
		errMessage: "error reattach cdrom: 'reattach_cdroms' should be between 1 and 4. if set to 0, `reattach_cdroms` is ignored and the step is skipped",
	})

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test case: %s", tt.name)

			driver := tt.vmMock
			state := new(multistep.BasicStateBag)
			state.Put("vm", driver)

			// Add a packer.Ui to the state
			state.Put("ui", &packer.BasicUi{
				Reader: os.Stdin,
				Writer: os.Stdout,
			})

			action := tt.step.Run(context.Background(), state)

			t.Logf("Expected action: %v, Actual action: %v", tt.expectedAction, action)

			// Print the error message from the state bag
			if err, ok := state.GetOk("error"); ok {
				t.Logf("Error: %v", err)
			}

			if action != tt.expectedAction {
				t.Fatalf("expected action %v, but got %v", tt.expectedAction, action)
			}
		})
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			state := basicStateBag(nil)
			state.Put("vm", c.vmMock)

			if action := c.step.Run(context.TODO(), state); action != c.expectedAction {
				t.Fatalf("unexpected action %v", action)
			}
			err, ok := state.Get("error").(error)
			if ok {
				if err.Error() != c.errMessage {
					t.Fatalf("unexpected error %s", err.Error())
				}
			} else {
				if c.fail {
					t.Fatalf("expected to fail but it didn't")
				}
			}

			if diff := cmp.Diff(c.vmMock, c.expectedVmMock,
				cmpopts.IgnoreInterfaces(struct{ error }{})); diff != "" {
				t.Fatalf("unexpected VirtualMachine calls: %s", diff)
			}
		})
	}
}
