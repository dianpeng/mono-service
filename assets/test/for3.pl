fn test1() {
  let i = 0;
  for {
    i++;
    if i >= 10 {
      break;
    }
  }
  assert::eq(i, 10);
}

fn test2() {
  let i = 0;
  for ;; {
    i++;
    if i >= 10 {
      break;
    }
  }
  assert::eq(i, 10);
}

fn test3() {
  let i = 0;
  for ;; {
    i++;
    if i >= 10 {
      break;
    }
    if i >= 20 {
      i = 0;
    }
  }
  assert::eq(i, 10);
}

fn test4() {
  let o = 0;
  for let i = 0; i < 10; i++ {
    o++;
    if i >= 1 {
      break;
    }
  }
  assert::eq(o, 2);
}

test {
  test1();
  test2();
  test3();
  test4();
}
