// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"reflect"
	"testing"
)

type user struct {
	Id    int
	Tag   string `valid:"Maxx(aa)"`
	Name  string `valid:"Required;"`
	Age   int    `valid:"Required;Range(1, 140)"`
	match string `valid:"Required; Match(/^(test)?\\w*@(/test/);com$/);Max(2)"`
}

// 测试获取validFuncs
func TestGetValidFuncs(t *testing.T) {
	u := user{Name: "test", Age: 1}
	tf := reflect.TypeOf(u)
	var vfs []ValidFunc
	var err error

	// 1. Id没有ValidFunc
	f, _ := tf.FieldByName("Id")
	if vfs, err = getValidFuncs(f); err != nil {
		t.Fatal(err)
	}
	if len(vfs) != 0 {
		t.Fatal("should get none ValidFunc")
	}

	// 2. Tag: 指定了错误的Func
	f, _ = tf.FieldByName("Tag")
	if vfs, err = getValidFuncs(f); err.Error() != "doesn't exsits Maxx valid function" {
		t.Fatal(err)
	}

	// 3. Name
	f, _ = tf.FieldByName("Name")
	if vfs, err = getValidFuncs(f); err != nil {
		t.Fatal(err)
	}
	if len(vfs) != 1 {
		t.Fatal("should get 1 ValidFunc")
	}
	if vfs[0].Name != "Required" && len(vfs[0].Params) != 0 {
		t.Error("Required funcs should be got")
	}

	// 4. Age
	f, _ = tf.FieldByName("Age")
	if vfs, err = getValidFuncs(f); err != nil {
		t.Fatal(err)
	}
	if len(vfs) != 2 {
		t.Fatal("should get 2 ValidFunc")
	}
	if vfs[0].Name != "Required" && len(vfs[0].Params) != 0 {
		t.Error("Required funcs should be got")
	}
	if vfs[1].Name != "Range" && len(vfs[1].Params) != 2 {
		t.Error("Range funcs should be got")
	}

	f, _ = tf.FieldByName("match")
	if vfs, err = getValidFuncs(f); err != nil {
		t.Fatal(err)
	}
	if len(vfs) != 3 {
		t.Fatal("should get 3 ValidFunc but now is", len(vfs))
	}
}

//
// 测试验证函数
//
func TestCall(t *testing.T) {
	u := user{Name: "test", Age: 180}
	tf := reflect.TypeOf(u)
	var vfs []ValidFunc
	var err error
	f, _ := tf.FieldByName("Age")
	if vfs, err = getValidFuncs(f); err != nil {
		t.Fatal(err)
	}

	// 获取到vfs之后如何执行呢?
	valid := &Validation{}
	// 构建完整的参数
	vfs[1].Params = append([]interface{}{valid, u.Age}, vfs[1].Params...)

	// 执行validator
	if _, err = funcs.Call(vfs[1].Name, vfs[1].Params...); err != nil {
		t.Fatal(err)
	}
	if len(valid.Errors) != 1 {
		t.Error("age out of range should be has an error")
	}
}
