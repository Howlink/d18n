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
	"testing"

	"d18n/common"
)

func init() {
	var err error
	common.InitTestEnv()

	common.Cfg.Schema = common.TestPath + "/test/schema.txt"
	emportStatus.Header, err = common.ParseSchema()
	if err != nil {
		panic(err.Error())
	}
}

func TestEmportCSV(t *testing.T) {
	orgCfg := common.Cfg

	common.Cfg.File = common.TestPath + "/test/actor.csv"
	common.Cfg.User = ""
	common.Cfg.Limit = 2
	common.Cfg.Table = "actor_new"
	common.Cfg.Replace = true
	common.Cfg.Comma = ','

	conn, _ := common.NewConnection()
	err := emportCSV(conn)
	if err != nil {
		t.Error(err.Error())
	}

	common.Cfg = orgCfg

}
