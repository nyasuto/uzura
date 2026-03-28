package wpt

// harnessJS is a minimal implementation of the WPT testharness.js API.
// It implements the subset of the API that Uzura can execute:
// test(), async_test(), assert_*, setup(), done(), add_completion_callback().
//
// Results are collected in the global __wpt_results__ object.
const harnessJS = `
(function() {
    "use strict";

    var results = {
        status: 0,  // 0=OK, 1=ERROR, 2=TIMEOUT
        message: "",
        tests: []
    };

    var completionCallbacks = [];
    var completed = false;
    var pendingAsync = 0;
    var explicitDone = false;

    // --- Assertion helpers ---

    function AssertionError(msg) {
        this.message = msg;
        this.name = "AssertionError";
    }
    AssertionError.prototype = Object.create(Error.prototype);

    function formatValue(v) {
        if (v === null) return "null";
        if (v === undefined) return "undefined";
        if (typeof v === "string") return JSON.stringify(v);
        if (typeof v === "object") {
            try { return JSON.stringify(v); } catch(e) { return String(v); }
        }
        return String(v);
    }

    function assert(cond, desc, msg) {
        if (!cond) {
            var full = msg || "Assertion failed";
            if (desc) full = desc + ": " + full;
            throw new AssertionError(full);
        }
    }

    // --- Public assertion API ---

    window.assert_true = function(val, desc) {
        assert(val === true, desc, "expected true got " + formatValue(val));
    };

    window.assert_false = function(val, desc) {
        assert(val === false, desc, "expected false got " + formatValue(val));
    };

    window.assert_equals = function(actual, expected, desc) {
        if (actual !== expected) {
            var msg = "expected " + formatValue(expected) + " but got " + formatValue(actual);
            assert(false, desc, msg);
        }
    };

    window.assert_not_equals = function(actual, expected, desc) {
        assert(actual !== expected, desc,
            "got disallowed value " + formatValue(actual));
    };

    window.assert_in_array = function(val, arr, desc) {
        assert(arr.indexOf(val) !== -1, desc,
            formatValue(val) + " not in " + formatValue(arr));
    };

    window.assert_array_equals = function(actual, expected, desc) {
        assert(Array.isArray(actual), desc, "value is not an array");
        assert(actual.length === expected.length, desc,
            "lengths differ: " + actual.length + " vs " + expected.length);
        for (var i = 0; i < actual.length; i++) {
            if (actual[i] !== expected[i]) {
                assert(false, desc,
                    "at index " + i + ": expected " + formatValue(expected[i]) +
                    " but got " + formatValue(actual[i]));
            }
        }
    };

    window.assert_class_string = function(obj, expected, desc) {
        var actual = Object.prototype.toString.call(obj);
        var exp = "[object " + expected + "]";
        assert(actual === exp, desc,
            "expected " + formatValue(exp) + " but got " + formatValue(actual));
    };

    window.assert_readonly = function(obj, prop, desc) {
        var d = Object.getOwnPropertyDescriptor(obj, prop);
        assert(d && !d.writable && !d.set, desc,
            prop + " is not readonly");
    };

    window.assert_throws_js = function(constructor, func, desc) {
        var threw = false;
        try { func(); } catch(e) {
            threw = true;
            assert(e instanceof constructor, desc,
                "threw " + e.name + " instead of " + (constructor.name || constructor));
        }
        assert(threw, desc, "expected exception but none thrown");
    };

    window.assert_throws_dom = function(type, func, desc) {
        var threw = false;
        try { func(); } catch(e) {
            threw = true;
            // Accept DOMException with matching name or code
            if (typeof type === "number") {
                assert(e.code === type, desc,
                    "expected code " + type + " but got " + e.code);
            } else {
                assert(e.name === type || e.message.indexOf(type) !== -1, desc,
                    "expected " + type + " but got " + (e.name || e.message));
            }
        }
        assert(threw, desc, "expected DOMException but none thrown");
    };

    window.assert_throws_exactly = function(expected, func, desc) {
        var threw = false;
        try { func(); } catch(e) {
            threw = true;
            assert(e === expected, desc,
                "threw wrong value: " + formatValue(e));
        }
        assert(threw, desc, "expected exception but none thrown");
    };

    window.assert_unreached = function(desc) {
        assert(false, desc, "reached unreachable code");
    };

    window.assert_regexp_match = function(actual, expected, desc) {
        assert(expected.test(actual), desc,
            formatValue(actual) + " did not match " + expected);
    };

    window.assert_own_property = function(obj, prop, desc) {
        assert(obj.hasOwnProperty(prop), desc,
            "missing own property " + formatValue(prop));
    };

    window.assert_inherits = function(obj, prop, desc) {
        assert(prop in obj, desc,
            "missing property " + formatValue(prop));
        assert(!obj.hasOwnProperty(prop), desc,
            formatValue(prop) + " is own property, not inherited");
    };

    // --- Test functions ---

    window.test = function(func, name, properties) {
        var t = {
            name: name || "(unnamed)",
            status: 0,  // 0=PASS
            message: ""
        };
        try {
            func.call(t);
        } catch(e) {
            t.status = 1;  // FAIL
            t.message = e.message || String(e);
        }
        results.tests.push(t);
    };

    window.async_test = function(nameOrFunc, name) {
        // async_test(name) or async_test(func, name)
        var testName;
        var setupFunc;
        if (typeof nameOrFunc === "function") {
            setupFunc = nameOrFunc;
            testName = name || "(unnamed async)";
        } else {
            testName = nameOrFunc || "(unnamed async)";
            setupFunc = null;
        }

        pendingAsync++;
        var t = {
            name: testName,
            status: 0,
            message: "",
            _done: false
        };

        t.step = function(func) {
            if (t._done) return;
            try {
                func.call(t);
            } catch(e) {
                t.status = 1;
                t.message = e.message || String(e);
                t.done();
            }
        };

        t.step_func = function(func) {
            return function() {
                t.step(function() { func.apply(t, arguments); });
            };
        };

        t.step_func_done = function(func) {
            return function() {
                t.step(function() {
                    if (func) func.apply(t, arguments);
                });
                t.done();
            };
        };

        t.unreached_func = function(desc) {
            return function() {
                t.step(function() { assert_unreached(desc); });
            };
        };

        t.done = function() {
            if (t._done) return;
            t._done = true;
            results.tests.push(t);
            pendingAsync--;
            checkComplete();
        };

        if (setupFunc) {
            try {
                setupFunc(t);
            } catch(e) {
                t.status = 1;
                t.message = e.message || String(e);
                if (!t._done) t.done();
            }
        }

        return t;
    };

    window.promise_test = function(func, name) {
        // Simplified: run as sync since goja has limited Promise support.
        var t = {
            name: name || "(unnamed promise)",
            status: 0,
            message: ""
        };
        try {
            var result = func(t);
            // If it returns a thenable, we can't really await it in goja.
            if (result && typeof result.then === "function") {
                t.status = 2; // NOTRUN — can't resolve promises
                t.message = "promise_test not fully supported";
            }
        } catch(e) {
            t.status = 1;
            t.message = e.message || String(e);
        }
        results.tests.push(t);
    };

    // --- Setup / completion ---

    window.setup = function(funcOrProps, maybeProps) {
        if (typeof funcOrProps === "function") {
            funcOrProps();
            if (maybeProps && maybeProps.explicit_done) explicitDone = true;
        } else if (funcOrProps) {
            if (funcOrProps.explicit_done) explicitDone = true;
        }
    };

    window.done = function() {
        checkComplete();
    };

    window.add_completion_callback = function(func) {
        completionCallbacks.push(func);
    };

    function checkComplete() {
        if (completed) return;
        if (pendingAsync > 0) return;
        completed = true;
        for (var i = 0; i < completionCallbacks.length; i++) {
            try {
                completionCallbacks[i](results.tests, results);
            } catch(e) {
                // ignore callback errors
            }
        }
    }

    // Internal: called by Go runner after script execution.
    window.__wpt_force_complete__ = function() {
        // Mark remaining async tests as timeout.
        if (!completed && pendingAsync > 0) {
            results.status = 2; // TIMEOUT
            pendingAsync = 0;
            checkComplete();
        } else if (!completed) {
            checkComplete();
        }
    };

    window.__wpt_results__ = results;
})();
`
