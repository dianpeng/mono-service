fn test1() {
  let v = 0;
  if v < 0 {
    v = 1;
  } elif v > 0 {
    v = 2;
  } else {
    v = 3;
  }
  assert::eq(v, 3);
}

fn test2() {
  let v = 0;
  if v < 0 {
    v =20;
  } elif v >= 0 {
    v = 1;
  } else {
    v = 10;
  }
  assert::eq(v, 1);
}

test {
  test1();
  test2();
}
