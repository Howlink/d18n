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

package save

import (
	"database/sql"
	"encoding/csv"
	"os"
	"strings"

	"github.com/LianjiaTech/d18n/common"
	"github.com/olekukonko/tablewriter"
)

// saveRows2CSV save rows result into csv format file
func saveRows2CSV(s *SaveStruct, rows *sql.Rows) error {
	var err error
	var file *os.File
	if strings.EqualFold(s.Config.File, "stdout") {
		file = os.Stdout
	} else {
		file, err = os.Create(s.Config.File)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	// 兼容 Windows 系统，文件头写入 UTF8 BOM，防止中文乱码。
	// windows 环境下导出的 csv 文件默认添加 UTF8 BOM。
	// 添加 BOM 对 less, awk 等 *nix 系统命令并不友好，因此仅对特定的文件名生效。
	// Linux 环境删除文件 UTF8 BOM 头命令：dos2unix xxx.csv
	if s.Config.BOM {
		_, err = file.WriteString(common.UTF8BOM)
		if err != nil {
			return err
		}
	}

	w := csv.NewWriter(file)
	w.Comma = s.Config.Comma
	defer w.Flush()

	// column info
	columnNames, err := rows.Columns()
	if err != nil {
		return err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	// set table header with column name
	if !s.Config.NoHeader {
		if s.Config.AutoFormatHeader {
			for i, v := range columnNames {
				columnNames[i] = tablewriter.Title(v)
			}
		}
		if err = w.Write(columnNames); err != nil {
			return err
		}
	}

	// init columns
	columnValues := make([]interface{}, len(columnNames))
	cols := make([]interface{}, len(columnNames))
	for j := range columnValues {
		cols[j] = &columnValues[j]
	}

	// set every table rows
	for rows.Next() {
		if !s.sample() {
			continue
		}
		s.Status.Lines++
		// limit return rows
		if s.Config.Limit != 0 && s.Status.Lines > s.Config.Limit {
			s.Status.Lines = s.Config.Limit
			break
		}

		// scan columns
		if err := rows.Scan(cols...); err != nil {
			return err
		}

		values := make([]string, len(columnNames))
		for j, col := range columnValues {
			if col == nil {
				values[j] = s.Config.NULLString
			} else {
				values[j] = s.String(col, columnTypes[j])

				// data mask
				values[j], err = s.Masker.Mask(s.FieldName(j), values[j])
				if err != nil {
					return err
				}

				// hex-blob
				values[j], _ = s.Config.Hex(s.FieldName(j), values[j])
			}
		}

		w.Write(values)
	}

	err = rows.Err()

	return err
}
