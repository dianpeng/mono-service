
// basic fors
fn test1() {
  let o = 0;
  for let j = 0; j < 100; j++ {
    o++;
  }
  assert::eq(o, 100);
}

fn test2() {
  let o = 0;
  for let j = 0; j < 10; j++ {
    for let k = 0; k < 10; k++ {
      for let m = 0; m < 10; m++ {
        for let q = 0; q < 10; q++ {
          o++;
        }
      }
    }
  }
  assert::eq(o, 10000);
}

// condition loop
fn test3() {
  let o = 0;
  for o < 100 {
    o++;
  }
  assert::eq(o, 100);
}

test {
  test1();
  test2();
  test3();
}
