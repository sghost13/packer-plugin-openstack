// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package openstack

import (
	"errors"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

// WaitForVolume waits for the given volume to become available.
func WaitForVolume(blockStorageClient *gophercloud.ServiceClient, volumeID string) error {
	maxNumErrors := 10
	numErrors := 0

	for {
		status, bootable, err := GetVolumeState(blockStorageClient, volumeID)
		if err != nil {
			errCode, ok := err.(*gophercloud.ErrUnexpectedResponseCode)
			if ok && (errCode.Actual == 500 || errCode.Actual == 404) {
				numErrors++
				if numErrors >= maxNumErrors {
					log.Printf("[ERROR] Maximum number of errors (%d) reached; failing with: %s", numErrors, err)
					return err
				}
				log.Printf("[ERROR] %d error received, will ignore and retry: %s", errCode.Actual, err)
				time.Sleep(2 * time.Second)
				continue
			}

			return err
		}

		if status == "available" && bootable {
			return nil
		}

		if status == "error" {
			return errors.New("The status of volume is error")
		}

		log.Printf("Waiting for volume creation status: %s (bootable: %t)", status, bootable)
		time.Sleep(2 * time.Second)
	}
}

// GetVolumeSize returns volume size in gigabytes based on the image min disk
// value if it's not empty.
// Or it calculates needed gigabytes size from the image bytes size.
func GetVolumeSize(imageClient *gophercloud.ServiceClient, imageID string) (int, error) {
	sourceImage, err := images.Get(imageClient, imageID).Extract()
	if err != nil {
		return 0, err
	}

	if sourceImage.MinDiskGigabytes != 0 {
		return sourceImage.MinDiskGigabytes, nil
	}

	volumeSizeMB := sourceImage.SizeBytes / 1024 / 1024
	volumeSizeGB := int(sourceImage.SizeBytes / 1024 / 1024 / 1024)

	// Increment gigabytes size if the initial size can't be divided without
	// remainder.
	if volumeSizeMB%1024 > 0 {
		volumeSizeGB++
	}

	return volumeSizeGB, nil
}

func GetVolumeState(blockStorageClient *gophercloud.ServiceClient, volumeID string) (string, bool, error) {
	volume, err := volumes.Get(blockStorageClient, volumeID).Extract()
	if err != nil {
		return "", false, err
	}

	return volume.Status, isVolumeBootable(volume.Bootable), nil
}

func isVolumeBootable(value string) bool {
	return value == "true"
}
