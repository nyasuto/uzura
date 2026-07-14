package js

import (
	"io"
	"testing"
)

func TestStorage_MethodsAndProperties(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindStorage(vm, NewMemStorage(), NewMemStorage())
	got, err := vm.Eval(`
		localStorage.setItem("a", "1");
		localStorage.b = "2";                    // プロパティ書き込み
		var r = [];
		r.push(localStorage.getItem("a"));       // "1"
		r.push(localStorage.a);                  // "1"  プロパティ読み出し
		r.push(localStorage.getItem("b"));       // "2"
		r.push(localStorage.length);             // 2
		r.push(localStorage.key(0));             // "a"
		r.push(localStorage.getItem("missing")); // null
		localStorage.removeItem("a");
		r.push(localStorage.length);             // 1
		localStorage.clear();
		r.push(localStorage.length);             // 0
		sessionStorage.setItem("s", "x");
		r.push(localStorage.getItem("s"));       // null (分離)
		JSON.stringify(r);
	`)
	if err != nil {
		t.Fatal(err)
	}
	want := `["1","1","2",2,"a",null,1,0,null]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStorage_OverwriteKeepsInsertionOrder(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindStorage(vm, NewMemStorage(), NewMemStorage())
	got, err := vm.Eval(`
		localStorage.setItem("a", "1");
		localStorage.setItem("b", "2");
		localStorage.setItem("a", "3"); // overwrite should not move "a" to the end
		var r = [];
		r.push(localStorage.key(0));
		r.push(localStorage.key(1));
		r.push(localStorage.getItem("a"));
		JSON.stringify(r);
	`)
	if err != nil {
		t.Fatal(err)
	}
	want := `["a","b","3"]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStorage_KeyOutOfRangeReturnsNull(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindStorage(vm, NewMemStorage(), NewMemStorage())
	got, err := vm.Eval(`
		localStorage.setItem("a", "1");
		JSON.stringify([localStorage.key(1), localStorage.key(-1)]);
	`)
	if err != nil {
		t.Fatal(err)
	}
	want := `[null,null]`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
