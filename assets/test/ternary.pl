/**
 * ternary expression testing
 */

fn test1() {
  assert::eq(1 if true else 2, 1);
  assert::eq(2 if false else 3, 3);

  assert::eq(if true {
    3;
  } else {
    1;
  }, 3);

  assert::eq(if false{
    3;
  } else {
    1;
  }, 1);
}

fn test2() {
  let v = 10;
  assert::eq(
    if v % 2 != 0 {
      foo();
      bar();
      coo();
    } else {
      try foo() else 10;
    },
  10);
}

test {
  test1();
  test2();
}
