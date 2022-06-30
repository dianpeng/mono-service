fn script_foo() {
  return "script_foo";
}

fn test1() {
  {
    let x = script_foo;
    assert::eq(x(), "script_foo");
  }
  {
    let x = callback;
    assert::eq(x(script_foo), "script_foo");
  }
  {
    let x = str::to_upper;
    assert::eq(x("a"), "A");
  }
  {
    let v = {};
    let x = v:set;
    assert::eq(x("a", "b"), {"a": "b"});
  }
}

test {
  test1();
}
