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

package packages

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
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

	styleRed, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#EC1C1C",
		},
	})

	styleGreen, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#3FA91E",
		},
	})
	columns := map[string]string{
		"A": "Package Name",
		"B": "Kinds \n (Suggestion API(s) usage)",
		"C": "Is using Webhooks",
		"D": "Multiple Architectures used",
		"E": "Has Scorecard Suggestions",
		"F": "Has Scorecard Falling Tests",
		"G": "Has Validator Errors",
		"H": "Has Validator Warnings",
		"I": "Has Invalid Versioning",
		"J": "Has Invalid SkipRange",
		"K": "Is multi-channel",
		"L": "Has Support for All Namespaces",
		"M": "Has Support for Single Namespaces",
		"N": "Has Support for Own Namespaces",
		"O": "Has Support for Multi Namespaces",
		"P": "Has Infrastructure Support",
		"Q": "Has possible performance issues",
		"R": "Has custom Scorecards",
		"S": "Issues (To process this report)",
	}

	// Header
	dt := time.Now().Format("2006-01-02")
	_ = f.SetCellValue(sheetName, "A1",
		fmt.Sprintf("Audit Packages Report (Generated at %s). IMPORTANT: This report only checks the head "+
			"operators of the channels. Use the bundles report to check all bundles", dt))
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
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", line), v.KindsDeprecateAPIs); err != nil {
			log.Errorf("to add KindsDeprecateAPIs cell value: %s", err)
		}
		if len(v.KindsDeprecateAPIs) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("B%d", line),
				fmt.Sprintf("B%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", line),
			pkg.GetYesOrNo(v.HasWebhooks)); err != nil {
			log.Errorf("to add HasWebhooks cell value: %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", line),
			pkg.GetFormatArrayWithBreakLine(v.MultipleArchitectures)); err != nil {
			log.Errorf("to add MultipleArchitectures cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", line),
			pkg.GetYesOrNo(v.HasScorecardSuggestions)); err != nil {
			log.Errorf("to add HasScorecardSuggestions cell value: %s", err)
		}

		if v.HasScorecardSuggestions {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("E%d", line),
				fmt.Sprintf("E%d", line), styleOrange)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", line),
			pkg.GetYesOrNo(v.HasScorecardFailingTests)); err != nil {
			log.Errorf("to add HasScorecardFailingTests cell value: %s", err)
		}

		if len(v.ScorecardFailingTests) > 0 {
			if err := f.AddComment(sheetName, fmt.Sprintf("F%d", line),
				fmt.Sprintf(`{"author":"Audit: ","text":"%s"}`, v.ScorecardFailingTests)); err != nil {
				log.Errorf("to add comment for ScorecardFailingTests: %s", err)
			}
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("F%d", line),
				fmt.Sprintf("F%d", line), styleRed)
		} else {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("F%d", line),
				fmt.Sprintf("F%d", line), styleGreen)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("G%d", line),
			pkg.GetYesOrNo(v.HasValidatorErrors)); err != nil {
			log.Errorf("to add HasValidatorErrors cell value: %s", err)
		}

		if v.HasValidatorErrors {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("G%d", line),
				fmt.Sprintf("G%d", line), styleRed)
		} else {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("G%d", line),
				fmt.Sprintf("G%d", line), styleGreen)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("H%d", line),
			pkg.GetYesOrNo(v.HasValidatorWarnings)); err != nil {
			log.Errorf("to add HasValidatorWarnings cell value: %s", err)
		}

		if v.HasValidatorWarnings {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("H%d", line),
				fmt.Sprintf("H%d", line), styleOrange)
		} else {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("H%d", line),
				fmt.Sprintf("H%d", line), styleGreen)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("I%d", line),
			pkg.GetYesOrNo(v.HasInvalidVersioning)); err != nil {
			log.Errorf("to add HasInvalidVersioning cell value: %s", err)
		}

		if v.HasInvalidVersioning {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("I%d", line),
				fmt.Sprintf("I%d", line), styleOrange)
		} else {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("I%d", line),
				fmt.Sprintf("I%d", line), styleGreen)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("J%d", line),
			pkg.GetYesOrNo(v.HasInvalidSkipRange)); err != nil {
			log.Errorf("to add HasInvalidSkipRange cell value: %s", err)
		}
		if v.HasInvalidSkipRange {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("J%d", line),
				fmt.Sprintf("J%d", line), styleOrange)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("K%d", line),
			pkg.GetYesOrNo(v.IsMultiChannel)); err != nil {
			log.Errorf("to add IsMultiChannel cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("L%d", line),
			pkg.GetYesOrNo(v.HasSupportForAllNamespaces)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("M%d", line),
			pkg.GetYesOrNo(v.HasSupportForSingleNamespace)); err != nil {
			log.Errorf("to add HasSupportForMultiNamespaces cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("N%d", line),
			pkg.GetYesOrNo(v.HasSupportForOwnNamespaces)); err != nil {
			log.Errorf("to add HasSupportForMultiNamespaces cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("O%d", line),
			pkg.GetYesOrNo(v.HasSupportForMultiNamespaces)); err != nil {
			log.Errorf("to add HasSupportForMultiNamespaces cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("P%d", line),
			pkg.GetYesOrNo(v.HasInfraAnnotation)); err != nil {
			log.Errorf("to add HasInfraAnnotation cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("Q%d", line),
			pkg.GetYesOrNo(v.HasPossiblePerformIssues)); err != nil {
			log.Errorf("to add HasPossiblePerformIssues cell value: %s", err)
		}

		if v.HasPossiblePerformIssues {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("Q%d", line),
				fmt.Sprintf("Q%d", line), styleOrange)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("R%d", line),
			pkg.GetYesOrNo(v.HasCustomScorecardTests)); err != nil {
			log.Errorf("to add HasCustomScorecardTests cell value: %s", err)
		}

		if v.HasCustomScorecardTests {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("R%d", line),
				fmt.Sprintf("R%d", line), styleGreen)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("S%d", line), v.AuditErrors); err != nil {
			log.Errorf("to add AuditErrors cell value: %s", err)
		}

	}

	// Remove the scorecard columns when that is disable
	if r.Flags.DisableScorecard {
		if err := f.SetColVisible(sheetName, "E", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "F", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
	}

	// Remove the validators columns when that is disable
	if r.Flags.DisableValidators {
		if err := f.SetColVisible(sheetName, "G", false); err != nil {
			log.Errorf("unable to remove validator columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "H", false); err != nil {
			log.Errorf("unable to remove validator columns : %s", err)
		}
	}

	if err := f.AddTable(sheetName, "A5", "S5", pkg.TableFormat); err != nil {
		log.Errorf("to set table format : %s", err)
	}

	reportFilePath := filepath.Join(r.Flags.OutputPath,
		pkg.GetReportName(r.Flags.IndexImage, "packages", "xlsx"))

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

	const reportType = "package"
	return pkg.WriteJSON(data, r.Flags.IndexImage, r.Flags.OutputPath, reportType)
}
