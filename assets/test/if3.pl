fn test1() {
  let v = 0;
  if v == 0 {
    v = 10;
  } elif v >= 0 {
    v = 20;
  } elif v <= 0 {
    v = 30;
  }
  assert::eq(v, 10);
}

fn test2() {
  let v = 0;
  if v != 0 {
    v = 10;
  } elif v > 0 {
    v = 20;
  } elif v <= 0 {
    v = 30;
  }
  assert::eq(v, 30);
}


test {
  test1();
  test2();
}
