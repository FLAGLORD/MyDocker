package container

import (
	"MyDocker/util"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	RootURL       = "/home/zyc/testMyDocker"
	MntURL        = "/home/zyc/testMyDocker/mnt/%s"
	WriteLayerURL = "/home/zyc/testMyDocker/writelayer/%s"
	WorkURL       = "/home/zyc/testMyDocker/work/%s"
)

// NewWorkSpace create a overlayfs as container root workspace
func NewWorkSpace(volume string, imageName string, containerName string) error {
	if err := CreateReadOnlyLayer(imageName); err != nil {
		return err
	}
	if err := createWriteLayer(containerName); err != nil {
		return err
	}
	if err := CreateMountPoint(containerName, imageName); err != nil {
		return err
	}
	if volume != "" {
		volumeURLs := strings.Split(volume, ":")

		if len(volumeURLs) == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			if err := MountVolume(volumeURLs, containerName); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("volume parameter input is not correct")
		}
	}
	return nil
}

// CreateReadOnlyLayer decompressions tar image
func CreateReadOnlyLayer(imageName string) error {
	imageURL := path.Join(RootURL, imageName+".tar")
	imageUntarURL := path.Join(RootURL, imageName)
	if !util.PathExists(imageUntarURL) {
		if !util.PathExists(imageURL) {
			return fmt.Errorf("unknown image %s", imageName)
		}
		if err := os.MkdirAll(imageUntarURL, 0777); err != nil {
			return fmt.Errorf("createReadOnlyLayer mkdir %s failed: %v", imageUntarURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", imageURL, "-C", imageUntarURL).CombinedOutput(); err != nil {
			os.RemoveAll(imageUntarURL)
			return fmt.Errorf("createReadOnlyLayer untar %s failed: %v", imageURL, err)
		}
	}
	return nil
}

// Creare write layer for container
func createWriteLayer(containerName string) error {
	writeLayerURL := fmt.Sprintf(WriteLayerURL, containerName)
	if !util.PathExists(writeLayerURL) {
		if err := os.MkdirAll(writeLayerURL, 0777); err != nil {
			return fmt.Errorf("createWriteLayer mkdir %s failed: %v", writeLayerURL, err)
		}
	}

	// For overlayfs, it needs a empty work dir when mounting
	workURL := fmt.Sprintf(WorkURL, containerName)
	if !util.PathExists(workURL) {
		if err := os.MkdirAll(workURL, 0777); err != nil {
			return fmt.Errorf("createWriteLayer mkdir(workdir) %s failed: %v", workURL, err)
		}
	}
	return nil
}

func MountVolume(volumeURLs []string, containerName string) error {
	// 宿主机 volume 位置
	hostVolumeURL := volumeURLs[0]
	if !util.PathExists(hostVolumeURL) {
		if err := os.MkdirAll(hostVolumeURL, 0777); err != nil {
			return fmt.Errorf("mountVolume hostVolume mkdir %s failed: %v", hostVolumeURL, err)
		}
	}

	containerURL := volumeURLs[1]
	mntURL := fmt.Sprintf(MntURL, containerName)
	containerVolumeURL := path.Join(mntURL, containerURL)
	if err := os.MkdirAll(containerVolumeURL, 0777); err != nil {
		return fmt.Errorf("mountVolume containerVolume mkdir failed: %v", err)
	}

	cmd := exec.Command("mount", "--bind", hostVolumeURL, containerVolumeURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mountVolume mount bind failed: %v", err)
	}
	return nil
}

// CreateMountPoint creates root dir of container
func CreateMountPoint(containerName string, imageName string) error {
	mntURL := fmt.Sprintf(MntURL, containerName)
	if !util.PathExists(mntURL) {
		if err := os.MkdirAll(mntURL, 0777); err != nil {
			return fmt.Errorf("createMountPoint mkdir %s failed: %v", mntURL, err)
		}
	}

	writeLayerURL := fmt.Sprintf(WriteLayerURL, containerName)
	imageURL := path.Join(RootURL, imageName)
	workDirURL := fmt.Sprintf(WorkURL, containerName)
	lowerdir := "lowerdir=" + imageURL
	upperdir := "upperdir=" + writeLayerURL
	workdir := "workdir=" + workDirURL
	dirOption := strings.Join([]string{lowerdir, upperdir, workdir}, ",")
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirOption, mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("createMountPoint mount overlay failed: %v", err)
	}
	return nil
}

func DeleteWorkSpace(volume string, containerName string) {
	if volume != "" {
		volumeURLs := strings.Split(volume, ":")
		if len(volumeURLs) == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			if err := DeleteVolume(volumeURLs, containerName); err != nil {
				log.Error(err)
			}
		}
	}
	if err := DeleteMountPoint(containerName); err != nil {
		log.Error(err)
	}
	if err := DeleteWriteLayer(containerName); err != nil {
		log.Error(err)
	}
}

func DeleteVolume(volumeURLs []string, containerName string) error {
	containerURL := volumeURLs[1]
	mntURL := fmt.Sprintf(MntURL, containerName)
	containerVolumeURL := path.Join(mntURL, containerURL)

	cmd := exec.Command("umount", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deleteVolume umount %s failed: %v", volumeURLs[0], err)
	}
	return nil
}

func DeleteMountPoint(containerName string) error {
	// first umount
	mntURL := fmt.Sprintf(MntURL, containerName)
	cmd := exec.Command("umount", mntURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deleteMountPoint umount %s failed: %v", mntURL, err)
	}

	if err := os.RemoveAll(mntURL); err != nil {
		return fmt.Errorf("deleteMountPoint remove %s failed: %v", mntURL, err)
	}
	return nil
}

// DeleteWriteLayer deletes writeLayer of container
func DeleteWriteLayer(containerName string) error {
	writeURL := fmt.Sprintf(WriteLayerURL, containerName)
	if err := os.RemoveAll(writeURL); err != nil {
		return fmt.Errorf("deleteWriteLayer remove dir %s failed: %v", writeURL, err)
	}
	return nil
}
