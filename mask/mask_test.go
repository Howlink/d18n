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

package mask

import (
	"fmt"
	"testing"

	"d18n/common"
)

// TestMask test FakeXXXX
func TestMask(t *testing.T) {
	orgMask := common.MaskConfig

	common.MaskConfig = map[string]common.MaskRule{
		"col1": {
			MaskFunc: "fake",
			Args:     []string{"name"},
		},
		"col2": {
			MaskFunc: "smokeleft",
			Args:     []string{"3", "*"},
		},
	}

	cases := [][]interface{}{
		{"col", 1},
		{"col1", "hello world"},
		{"col2", "hello earth"},
	}

	for _, c := range cases {
		ret, err := Mask(c[0].(string), c[1])
		if err != nil {
			t.Error(common.MaskConfig[c[0].(string)], err.Error())
		}
		fmt.Println(c[0], common.MaskConfig[c[0].(string)], ret)
	}

	common.MaskConfig = orgMask
}

func ExampleMask() {
	orgMask := common.MaskConfig

	common.MaskConfig = map[string]common.MaskRule{
		"col1": {
			MaskFunc: "crc32",
			Args:     []string{},
		},
	}
	fmt.Println(Mask("col", "hello world"))
	fmt.Println(Mask("col1", "hello world"))
	// Output:
	// hello world <nil>
	// 0d4a1185 <nil>

	common.MaskConfig = orgMask
}
