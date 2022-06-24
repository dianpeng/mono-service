/**
 ** precedence
 ** 1. ||
 ** 2. &&
 ** 3. == != ~ !~
 ** 4. < <= > >=
 ** 5. ...
 **/

fn testPrecedence() {
  assert::eq(false || false && false, false);
  assert::yes(1 == 1 && 2 != 1);
  assert::yes(1 != 1 || 2 == 2);
  assert::yes(1 < 2 && 2 >= 2);
  assert::no(1 != 1 || 2 != 2);
}

// our || and && operators will just leave the value as is, without converting
// them to boolean, so it maybe useful for user to get value from it
fn testSideEffect() {
  assert::eq(false || {}, {});
  assert::eq(100 || [], 100);
  assert::eq(true && "Hello World", "Hello World");
  assert::eq(null || 100, 100);
}

fn test2Boolean() {
  assert::no(0);
  assert::yes(1);
  assert::no(0.0);
  assert::yes(1.0);
  assert::no("");
  assert::yes("A");
  assert::no(false);
  assert::yes(true);
  assert::no([]);
  assert::no({});
  assert::yes(fn(): 1);
  assert::yes(r""); // this is a regexp
  assert::yes((null, null)); // this is a pair, must be true
  assert::no(null);
}

test {
  testPrecedence();
  testSideEffect();
  test2Boolean();
}
