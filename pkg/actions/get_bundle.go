// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	log "github.com/sirupsen/logrus"
)

// Manifest define the manifest.json which is  required to read the bundle
type Manifest struct {
	Config string
	Layers []string
}

// GetDataFromBundleImage returns the bundle from the image
func GetDataFromBundleImage(auditBundle *models.AuditBundle,
	disableScorecard, disableValidators, serverMode bool, label, labelValue string) *models.AuditBundle {

	if len(auditBundle.OperatorBundleImagePath) < 1 {
		auditBundle.Errors = append(auditBundle.Errors,
			errors.New("not found bundle path stored in the index.db").Error())
		return auditBundle
	}

	err := DownloadImage(auditBundle.OperatorBundleImagePath)
	if err != nil {
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to download container image (%s): %s", auditBundle.OperatorBundleImagePath, err).Error())
		return auditBundle
	}

	bundleDir := createBundleDir(auditBundle)
	extractBundleFromImage(auditBundle, bundleDir)

	inspectManifest, err := pkg.RunDockerInspect(auditBundle.OperatorBundleImagePath)
	if err != nil {
		auditBundle.Errors = append(auditBundle.Errors, err.Error())
	} else {
		// Gathering data by inspecting the operator bundle image
		if len(label) > 0 {
			value := inspectManifest.DockerConfig.Labels[label]
			if value == labelValue {
				auditBundle.FoundLabel = true
			}
		}

		// 4.8 images has note the build-date in the label
		if len(inspectManifest.Created) > 0 {
			auditBundle.BuildAt = inspectManifest.Created
		} else {
			auditBundle.BuildAt = inspectManifest.DockerConfig.Labels["build-date"]
		}

		auditBundle.OCPLabel = inspectManifest.DockerConfig.Labels["com.redhat.openshift.versions"]
	}

	// Read the bundle
	auditBundle.Bundle, err = apimanifests.GetBundleFromDir(filepath.Join(bundleDir, "bundle"))
	if err != nil {
		auditBundle.Errors = append(auditBundle.Errors, fmt.Errorf("unable to get the bundle: %s", err).Error())
		return auditBundle
	}

	// Gathering data from scorecard
	if !disableScorecard {
		auditBundle = RunScorecard(filepath.Join(bundleDir, "bundle"), auditBundle)
	}

	// Run validators
	if !disableValidators {
		auditBundle = RunValidators(auditBundle)

	}

	cleanupBundleDir(auditBundle, bundleDir, serverMode)

	return auditBundle
}

func createBundleDir(auditBundle *models.AuditBundle) string {
	dir := fmt.Sprintf("./tmp/%s", auditBundle.OperatorBundleName)
	cmd := exec.Command("mkdir", dir)
	_, err := pkg.RunCommand(cmd)
	if err != nil {
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to create the dir for the bundle: %s", err).Error())
	}
	return dir
}

func extractBundleFromImage(auditBundle *models.AuditBundle, bundleDir string) {
	imageName := strings.Split(auditBundle.OperatorBundleImagePath, "@")[0]
	tarPath := fmt.Sprintf("%s/%s.tar", bundleDir, auditBundle.OperatorBundleName)
	cmd := exec.Command("docker", "save", imageName, "-o", tarPath)
	_, err := pkg.RunCommand(cmd)
	if err != nil {
		log.Errorf("unable to save the bundle image : %s", err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to save the bundle image : %s", err).Error())
	}

	cmd = exec.Command("tar", "-xvf", tarPath, "-C", bundleDir)
	_, err = pkg.RunCommand(cmd)
	if err != nil {
		log.Errorf("unable to untar the bundle image: %s", err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to untar the bundle image : %s", err).Error())
	}

	cmd = exec.Command("mkdir", filepath.Join(bundleDir, "bundle"))
	_, err = pkg.RunCommand(cmd)
	if err != nil {
		log.Errorf("error to create the bundle bundleDir: %s", err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("error to create the bundle bundleDir : %s", err).Error())
	}

	bundleConfigFilePath := filepath.Join(bundleDir, "manifest.json")
	existingFile, err := ioutil.ReadFile(bundleConfigFilePath)
	if err == nil {
		var bundleLayerConfig []Manifest
		if err := json.Unmarshal(existingFile, &bundleLayerConfig); err != nil {
			log.Errorf("unable to Unmarshal manifest.json: %s", err)
			auditBundle.Errors = append(auditBundle.Errors,
				fmt.Errorf("unable to Unmarshal manifest.json: %s", err).Error())
		}
		if bundleLayerConfig == nil {
			log.Errorf("error to untar layers")
			auditBundle.Errors = append(auditBundle.Errors,
				fmt.Errorf("error to untar layers: %s", err).Error())
		}

		for _, layer := range bundleLayerConfig[0].Layers {
			cmd = exec.Command("tar", "-xvf", filepath.Join(bundleDir, layer), "-C", filepath.Join(bundleDir, "bundle"))
			_, err = pkg.RunCommand(cmd)
			if err != nil {
				log.Errorf("unable to untar layer : %s", err)
				auditBundle.Errors = append(auditBundle.Errors,
					fmt.Errorf("error to untar layers : %s", err).Error())
			}
		}
	} else {
		// If the docker manifest was not found then check if has just one layer
		cmd = exec.Command("tar", "-xvf", fmt.Sprintf("%s/layer.tar", bundleDir), "-C", filepath.Join(bundleDir, "bundle"))
		_, err = pkg.RunCommand(cmd)
		if err != nil {
			log.Errorf("unable to untar layer : %s", err)
			auditBundle.Errors = append(auditBundle.Errors,
				fmt.Errorf("unable to untar layer: %s", err).Error())
		}
	}

	// Remove files in the image to allow load the bundle
	cmd = exec.Command("rm", "-rf", fmt.Sprintf("%s/bundle/manifests/.wh..wh..opq", bundleDir))
	_, _ = pkg.RunCommand(cmd)

	cmd = exec.Command("rm", "-rf", fmt.Sprintf("%s/bundle/metadata/.wh..wh..opq", bundleDir))
	_, _ = pkg.RunCommand(cmd)

	cmd = exec.Command("rm", "-rf", fmt.Sprintf("%s/bundle/root/", bundleDir))
	_, _ = pkg.RunCommand(cmd)
}

func cleanupBundleDir(auditBundle *models.AuditBundle, dir string, serverMode bool) {
	cmd := exec.Command("rm", "-rf", dir)
	_, _ = pkg.RunCommand(cmd)

	if !serverMode {
		cmd = exec.Command("docker", "rmi", auditBundle.OperatorBundleImagePath)
		_, _ = pkg.RunCommand(cmd)
	}
}

func DownloadImage(image string) error {
	cmd := exec.Command("docker", "pull", image)
	_, err := pkg.RunCommand(cmd)
	return err
}
