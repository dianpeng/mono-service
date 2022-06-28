fn test1() {
  let v = 0;
  if v == 0 {
    v = 10;
  }
  assert::eq(v, 10);
}

fn test2() {
  let v = 0;
  if v != 0 {
    v = 10;
  }
  assert::eq(v, 0);
}

test {
  test1();
  test2();
}
