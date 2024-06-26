/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package fwprovider_test

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/bpg/terraform-provider-proxmox/fwprovider/test"
	"github.com/bpg/terraform-provider-proxmox/proxmox/helpers/ptr"
	"github.com/bpg/terraform-provider-proxmox/proxmox/nodes/storage"
)

const (
	fakeFileISO   = "https://cdn.githubraw.com/rafsaf/a4b19ea5e3485f8da6ca4acf46d09650/raw/d340ec3ddcef9b907ede02f64b5d3f694da5d081/fake_file.iso"
	fakeFileQCOW2 = "https://cdn.githubraw.com/rafsaf/036eece601975a3ad632a77fc2809046/raw/10500012fca9b4425b50de67a7258a12cba0c076/fake_file.qcow2"
)

func TestAccResourceDownloadFile(t *testing.T) {
	te := test.InitEnvironment(t)

	te.AddTemplateVars(map[string]interface{}{
		"FakeFileISO":   fakeFileISO,
		"FakeFileQCOW2": fakeFileQCOW2,
	})

	tests := []struct {
		name  string
		steps []resource.TestStep
	}{
		{"download qcow2 file", []resource.TestStep{{
			Config: te.RenderConfig(`
				resource "proxmox_virtual_environment_download_file" "qcow2_image" {
					content_type       = "iso"
					node_name          = "{{.NodeName}}"
					datastore_id       = "{{.DatastoreID}}"
					file_name          = "fake_qcow2_file.img"
					url                =  "{{.FakeFileQCOW2}}"
					checksum           = "688787d8ff144c502c7f5cffaafe2cc588d86079f9de88304c26b0cb99ce91c6"
					checksum_algorithm = "sha256"
					overwrite_unmanaged = true
				  }`),
			Check: resource.ComposeTestCheckFunc(
				test.ResourceAttributes("proxmox_virtual_environment_download_file.qcow2_image", map[string]string{
					"id":                 "local:iso/fake_qcow2_file.img",
					"content_type":       "iso",
					"node_name":          te.NodeName,
					"datastore_id":       te.DatastoreID,
					"url":                fakeFileQCOW2,
					"file_name":          "fake_qcow2_file.img",
					"upload_timeout":     "600",
					"size":               "3",
					"verify":             "true",
					"checksum":           "688787d8ff144c502c7f5cffaafe2cc588d86079f9de88304c26b0cb99ce91c6",
					"checksum_algorithm": "sha256",
				}),
				test.NoResourceAttributesSet("proxmox_virtual_environment_download_file.qcow2_image", []string{
					"decompression_algorithm",
				}),
			),
		}}},
		{"download & update iso file", []resource.TestStep{
			{
				Config: te.RenderConfig(`
				resource "proxmox_virtual_environment_download_file" "iso_image" {
					content_type = "iso"
					node_name    = "{{.NodeName}}"
					datastore_id = "{{.DatastoreID}}"
					url          = "{{.FakeFileISO}}"
					overwrite_unmanaged = true
				  }`),
				Check: resource.ComposeTestCheckFunc(
					test.ResourceAttributes("proxmox_virtual_environment_download_file.iso_image", map[string]string{
						"id":             "local:iso/fake_file.iso",
						"node_name":      te.NodeName,
						"datastore_id":   te.DatastoreID,
						"url":            fakeFileISO,
						"file_name":      "fake_file.iso",
						"upload_timeout": "600",
						"size":           "3",
						"verify":         "true",
					}),
					test.NoResourceAttributesSet("proxmox_virtual_environment_download_file.iso_image", []string{
						"checksum",
						"checksum_algorithm",
						"decompression_algorithm",
					}),
				),
			},
			{
				Config: te.RenderConfig(`
				resource "proxmox_virtual_environment_download_file" "iso_image" {
					content_type   = "iso"
					node_name      = "{{.NodeName}}"
					datastore_id   = "{{.DatastoreID}}"
					file_name      = "fake_iso_file.img"
					url            = "{{.FakeFileISO}}"
					upload_timeout = 10000
					overwrite_unmanaged = true
				  }`),
				Check: resource.ComposeTestCheckFunc(
					test.ResourceAttributes("proxmox_virtual_environment_download_file.iso_image", map[string]string{
						"id":             "local:iso/fake_iso_file.img",
						"content_type":   "iso",
						"node_name":      te.NodeName,
						"datastore_id":   te.DatastoreID,
						"url":            fakeFileISO,
						"file_name":      "fake_iso_file.img",
						"upload_timeout": "10000",
						"size":           "3",
						"verify":         "true",
					}),
					test.NoResourceAttributesSet("proxmox_virtual_environment_download_file.iso_image", []string{
						"checksum",
						"checksum_algorithm",
						"decompression_algorithm",
					}),
				),
			},
		}},
		{"override unmanaged file", []resource.TestStep{{
			PreConfig: func() {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				_ = te.NodeStorageClient().DeleteDatastoreFile(ctx, "iso/fake_file.iso") //nolint: errcheck

				err := te.NodeStorageClient().DownloadFileByURL(ctx, &storage.DownloadURLPostRequestBody{
					Content:  ptr.Ptr("iso"),
					FileName: ptr.Ptr("fake_file.iso"),
					Node:     ptr.Ptr(te.NodeName),
					Storage:  ptr.Ptr(te.DatastoreID),
					URL:      ptr.Ptr(fakeFileISO),
				})
				require.NoError(t, err)

				t.Cleanup(func() {
					e := te.NodeStorageClient().DeleteDatastoreFile(context.Background(), "iso/fake_file.iso")
					require.NoError(t, e)
				})
			},
			Config: te.RenderConfig(`
				resource "proxmox_virtual_environment_download_file" "iso_image3" {
					content_type        = "iso"
					node_name           = "{{.NodeName}}"
					datastore_id        = "{{.DatastoreID}}"
					url 		        = "{{.FakeFileISO}}"
					file_name           = "fake_iso_file3.iso"
					overwrite_unmanaged = true
				  }`),
			Check: resource.ComposeTestCheckFunc(
				test.ResourceAttributes("proxmox_virtual_environment_download_file.iso_image3", map[string]string{
					"id":           "local:iso/fake_iso_file3.iso",
					"content_type": "iso",
					"node_name":    te.NodeName,
					"datastore_id": te.DatastoreID,
					"url":          fakeFileISO,
					"file_name":    "fake_iso_file3.iso",
					"size":         "3",
					"verify":       "true",
				}),
				test.NoResourceAttributesSet("proxmox_virtual_environment_download_file.iso_image3", []string{
					"checksum",
					"checksum_algorithm",
					"decompression_algorithm",
				}),
			),
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: te.AccProviders,
				Steps:                    tt.steps,
			})
		})
	}
}
