fn xx(a, b, c) {
  return a + b + c;
}

fn test1() {
  assert::eq("HELLO", "hello" | str::to_upper);
  assert::eq("HELLO", "hello" | str::to_upper());
  assert::eq(10, 5 | xx(2, 3));
}

// universal calling style
fn test2() {
  let script_function = xx; // function symbol
  let local_function = fn(a, b, c) {
    return a * b * c;
  };
  let native_function = str::to_upper;
  let member_function = {}:set;

  assert::eq(
    type(script_function), "closure");
  assert::eq(
    type(local_function), "closure");
  assert::eq(
    type(native_function), "closure");
  assert::eq(
    type(member_function), "closure");

  assert::eq(
    1 | script_function(2, 3), 1+2+3);
  assert::eq(
    1 | local_function(2, 3), 1 * 2 * 3);
  assert::eq(
    "a" | native_function, "A");
  assert::eq(
    "a" | member_function("b"), {"a" : "b"});
}

test {
  test1();
  test2();
}
