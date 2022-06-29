fn test1() {
  let v = 0;
  let vv = if v != 0 {
    foo();
    bar();
    coo();
  } else {
    try foo() else 10;
  };

  assert::eq(vv, 10);
}

fn test2() {
  let v = 0;
  assert::eq(
    if v != 0 {
      foo();
      bar();
      coo();
    } else {
      try foo() else 10;
    },
    10
  );
}

fn test3() {
  let v = 10;
  assert::eq(
    if v % 2 != 0 {
      foo();
      bar();
      coo();
    } else {
      try foo() else 100;
    },
  100);
}

fn test4() {
  let v = 10;
  assert::eq(
    if v % 2 != 0 {
      foo();
      bar();
      coo();
    } else {
      try foo() else 10;
    },
  100);
}

test {
  test1();
  test2();
	test3();
  assert::eq(try test4() else 0, 0);
}
