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
	"fmt"
	"os"
	"strings"

	"d18n/common"
	"d18n/mask"
)

// saveRows2CSV save rows result into csv format file
func saveRows2CSV(rows *sql.Rows) error {
	var err error
	var file *os.File
	if strings.EqualFold(common.Cfg.File, "stdout") {
		file = os.Stdout
	} else {
		file, err = os.Create(common.Cfg.File)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	// 兼容 Windows 系统，文件头写入 UTF8 BOM，防止中文乱码。
	// windows 环境下导出的 csv 文件默认添加 UTF8 BOM。
	// 添加 BOM 对 less, awk 等 *nix 系统命令并不友好，因此仅对特定的文件名生效。
	// Linux 环境删除文件 UTF8 BOM 头命令：dos2unix xxx.csv
	if common.Cfg.BOM {
		_, err = file.WriteString(common.UTF8BOM)
		if err != nil {
			return err
		}
	}

	w := csv.NewWriter(file)
	w.Comma = common.Cfg.Comma
	defer w.Flush()

	// set table header with column name
	if !common.Cfg.NoHeader {
		err = w.Write(common.DBParserColumnNames(saveStatus.Header))
		if err != nil {
			return err
		}
	}

	// init columns
	columns := make([]interface{}, len(saveStatus.Header))
	cols := make([]interface{}, len(saveStatus.Header))
	for j := range columns {
		cols[j] = &columns[j]
	}

	// set every table rows
	for rows.Next() {
		saveStatus.Lines++
		// limit return rows
		if common.Cfg.Limit != 0 && saveStatus.Lines > common.Cfg.Limit {
			break
		}

		// scan columns
		if err := rows.Scan(cols...); err != nil {
			return err
		}

		values := make([]string, len(columns))
		for j, col := range columns {
			if col == nil {
				values[j] = common.Cfg.NULLString
			} else {
				switch col.(type) {
				case []byte:
					values[j] = string(col.([]byte))
				case []string:
					values[j] = common.ParseArray(col.([]string))
				default:
					values[j] = fmt.Sprint(col)
				}

				// data mask
				values[j], err = mask.Mask(saveStatus.Header[j].Name(), values[j])
				if err != nil {
					return err
				}

				// hex-blob
				values[j], _ = common.HexBLOB(saveStatus.Header[j].Name(), values[j])
			}
		}

		w.Write(values)
	}

	err = rows.Err()

	return err
}
