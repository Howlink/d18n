/*
 * Copyright(c)  2021 Lianjia, Inc.  All Rights Reserved
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package emport

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	xlsx "github.com/360EntSecGroup-Skylar/excelize/v2"
)

func emportXlsx(e *EmportStruct, conn *sql.DB) error {
	if strings.EqualFold(e.Config.File, "stdin") {
		return errors.New("xlsx not support stdin")
	}

	fd, err := xlsx.OpenFile(e.Config.File)
	if err != nil {
		return err
	}

	sheets := fd.GetSheetList()
	if len(sheets) > 0 {
		rows, err := fd.Rows(sheets[0])
		if err != nil {
			return err
		}

		insertPrefix, err := e.Config.SQLInsertPrefix(e.Config.DBParseHeaderColumn(e.Status.Header))
		if err != nil {
			return err
		}

		var sql string
		var sqlCounter int
		for rows.Next() {
			e.Status.Lines++
			// read row
			row, err := rows.Columns()
			if err != nil {
				return err
			}

			// skip header line
			if e.Status.Lines == 1 && !e.Config.NoHeader {
				continue
			}

			// SkipLines
			if e.Status.Lines <= e.Config.SkipLines {
				continue
			}
			if e.Config.Limit > 0 &&
				(e.Status.Lines-e.Config.SkipLines) > e.Config.Limit {
				e.Status.Lines = e.Config.Limit
				break
			}

			// ignore blank lines
			if e.Config.IgnoreBlank && len(row) == 0 {
				continue
			}

			// mask data
			if e.Config.IgnoreBlank {
				rowLen := len(row)
				// ignore extra blank cell
				if len(e.Status.Header) < len(row) {
					rowLen = len(e.Status.Header)
				}
				row, err = e.Masker.MaskRow(e.Status.Header, row[:rowLen])
			} else {
				row, err = e.Masker.MaskRow(e.Status.Header, row)
			}
			if err != nil {
				return err
			}

			// concat sql
			values, err := e.Config.SQLInsertValues(e.Status.Header, e.Config.DBParseNullString(e.Status.Header, row))
			if err != nil {
				return err
			}

			// extended-insert
			sqlCounter++
			sql += e.Config.SQLMultiValues(sqlCounter, insertPrefix, values)
			if e.Config.ExtendedInsert <= 1 || sqlCounter%e.Config.ExtendedInsert == 0 {
				line := 1
				if e.Config.ExtendedInsert > 1 {
					line = e.Config.ExtendedInsert
				}

				err = e.executeSQL(sql, conn)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error=%s; sql=%s", err.Error(), sql)
					sqlCounter -= line
					e.Status.ImportFailRows += line
					// return err
				}
				sql = ""
			}
		}

		// execute last SQL
		if sql != "" {
			err = e.executeSQL(sql, conn)
			if err != nil {
				line := 1
				if e.Config.ExtendedInsert > 1 {
					line = sqlCounter % e.Config.ExtendedInsert
				}
				fmt.Fprintf(os.Stderr, "error=%s; sql=%s", err.Error(), sql)
				sqlCounter -= line
				e.Status.ImportFailRows += line
			}
		}
		e.Status.Rows = sqlCounter

	} else {
		return fmt.Errorf("empty xlsx file")
	}
	return err
}
