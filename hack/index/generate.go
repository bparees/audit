// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this File except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This module is used to generate the index.html page
package main

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/audit/hack"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"
)

type DashboardPerCatalog struct {
	Name    string
	Reports []Reports
}

type Reports struct {
	Path string
	Name string
	Kind string
}

type Index struct {
	DashboardPerCatalog []DashboardPerCatalog
}

//nolint:gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fullReportsPath := filepath.Join(currentPath, hack.ReportsPath)

	dirs := map[string]string{
		"redhat_certified_operator_index": "registry.redhat.io/redhat/certified-operator-index",
		"redhat_community_operator_index": "registry.redhat.io/redhat/community-operator-index",
		"redhat_redhat_marketplace_index": "registry.redhat.io/redhat/redhat-marketplace-index",
		"redhat_redhat_operator_index":    "registry.redhat.io/redhat/redhat-operator-index",
		"operatorhubio_catalog":           "quay.io/operatorhubio/catalog",
	}

	var all []DashboardPerCatalog
	var index Index
	// nolint:scopelint
	for dir, image := range dirs {
		pathToWalk := filepath.Join(fullReportsPath, dir, "dashboards")

		if _, err := os.Stat(pathToWalk); err != nil && os.IsNotExist(err) {
			continue
		}

		dash := DashboardPerCatalog{Name: image}
		err = filepath.Walk(pathToWalk, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), "html") {
				var kind = "UNKNOWN"
				if strings.Contains(info.Name(), "deprecate") {
					kind = "Deprecated API(s) in 1.22/OCP 4.9"
				} else if strings.Contains(info.Name(), "grade") {
					kind = "Grade - Experimental"
				}
				tagValue := "latest"
				if strings.Contains(info.Name(), "v") {
					tagS := strings.Split(info.Name(), "v")[1]
					tagValue = strings.Split(tagS, "_")[0]
				}

				name := fmt.Sprintf("[%s] - Tag: %s", kind, tagValue)
				dash.Reports = append(dash.Reports,
					Reports{Path: filepath.Join(hack.ReportsPath, dir, "dashboards", info.Name()),
						Name: name, Kind: kind})
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
		sort.Slice(dash.Reports[:], func(i, j int) bool {
			return dash.Reports[i].Name < dash.Reports[j].Name
		})
		all = append(all, dash)

		sort.Slice(all[:], func(i, j int) bool {
			return all[i].Name < all[j].Name
		})
	}

	index.DashboardPerCatalog = all
	indexPath := filepath.Join(currentPath, "index.html")
	command := exec.Command("rm", "-rf", indexPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	f, err := os.Create(indexPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/index/template.go.tmpl")))
	err = t.Execute(f, index)
	if err != nil {
		panic(err)
	}

	f.Close()
}
