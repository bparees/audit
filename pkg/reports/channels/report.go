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

package channels

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"
)

type Report struct {
	Columns           []Column  `json:"columns"`
	Flags             BindFlags `json:"flags"`
	IndexImageInspect pkg.DockerInspectManifest
	GenerateAt        string
}

//todo: fix the complexity
//nolint:gocyclo
func (r *Report) writeXls() error {
	const sheetName = "Sheet1"
	f := excelize.NewFile()

	styleOrange, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#ec8f1c",
		},
	})

	columns := map[string]string{
		"A": "Package Name",
		"B": "Channel Name",
		"C": "Is using skips",
		"D": "Is using skipRange",
		"E": "Is Following Name Convention",
		"F": "Has Invalid Versioning",
		"G": "Has Invalid SkipRange",
		"H": "Issues (To process this report)",
	}

	// Header
	dt := time.Now().Format("2006-01-02")
	_ = f.SetCellValue(sheetName, "A1",
		fmt.Sprintf("Audit Channels Report (Generated at %s)", dt))
	_ = f.SetCellValue(sheetName, "A2", "Image used")
	_ = f.SetCellValue(sheetName, "B2", r.Flags.IndexImage)
	_ = f.SetCellValue(sheetName, "A3", "Image Index Create Date:")
	_ = f.SetCellValue(sheetName, "B3", r.IndexImageInspect.Created)
	_ = f.SetCellValue(sheetName, "A4", "Image Index ID:")
	_ = f.SetCellValue(sheetName, "B4", r.IndexImageInspect.ID)

	for k, v := range columns {
		_ = f.SetCellValue(sheetName, fmt.Sprintf("%s5", k), v)
	}

	for k, v := range r.Columns {
		line := k + 6

		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", line), v.PackageName); err != nil {
			log.Errorf("to add packageName cell value: %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", line), v.ChannelName); err != nil {
			log.Errorf("to add packageName cell value: %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", line), pkg.GetYesOrNo(v.IsUsingSkips)); err != nil {
			log.Errorf("to add packageName cell value: %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", line), pkg.GetYesOrNo(v.IsUsingSkipRange)); err != nil {
			log.Errorf("to add IsUsingSkipRange cell value: %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", line),
			pkg.GetYesOrNo(v.IsFollowingNameConvention)); err != nil {
			log.Errorf("to add HasIvalidVersioning cell value: %s", err)
		}
		if !v.IsFollowingNameConvention {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("E%d", line),
				fmt.Sprintf("E%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", line), pkg.GetYesOrNo(v.HasInvalidVersioning)); err != nil {
			log.Errorf("to add HasIvalidVersioning cell value: %s", err)
		}
		if v.HasInvalidVersioning {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("F%d", line),
				fmt.Sprintf("F%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("G%d", line), pkg.GetYesOrNo(v.HasInvalidSkipRange)); err != nil {
			log.Errorf("to add HasInvalidSkipRange cell value: %s", err)
		}
		if v.HasInvalidSkipRange {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("G%d", line),
				fmt.Sprintf("G%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("H%d", line), v.AuditErrors); err != nil {
			log.Errorf("to add AuditErrors cell value: %s", err)
		}

	}

	if err := f.AddTable(sheetName, "A5", "H5", pkg.TableFormat); err != nil {
		log.Errorf("to set table format : %s", err)
	}

	reportFilePath := filepath.Join(r.Flags.OutputPath,
		pkg.GetReportName(r.Flags.IndexImage, "channels", "xlsx"))

	if err := f.SaveAs(reportFilePath); err != nil {
		return err
	}
	return nil
}

func (r *Report) writeJSON() error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	const reportType = "channels"
	return pkg.WriteJSON(data, r.Flags.IndexImage, r.Flags.OutputPath, reportType)
}
