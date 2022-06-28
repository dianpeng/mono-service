fn test1() {
  let v = 0;
  if v == 0 {
    v = 10;
  } else {
    v = 30;
  }
  assert::eq(v, 10);
}

fn test2() {
  let v = 0;
  if v != 0 {
    v = 10;
  } else {
    v = 20;
  }
  assert::eq(v, 20);
}

test {
  test1();
  test2();
}
