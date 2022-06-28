fn test1() {
  let i = 0;
  for let j = 0; j < 10; j++ {
    i++;
    continue;
    j++;
  }
  assert::eq(i, 10);
}

fn test2() {
  let cnt = 0;
  let i = 0;
  for ; i < 10; i++ {
    if i % 2 == 0 {
      i++;
      continue;
    }
    cnt++;
  }
  assert::eq(cnt, 0);
}

fn test3() {
  let cnt = 0;
  let i = 0;
  for ; i < 10; i++ {
    if i % 2 == 1 {
      if i >= 6 {
        if i >= 8 {
          if i >= 9 {
            continue;
          }
        }
      }
    }
    cnt++;
  }
  assert::eq(cnt, 9);
}

fn test4() {
  for let i = 0; i < 10; i++ {
    if i >= 2 {
      if i >= 3 {
        if i >= 4 {
          if i >= 5 {
            if i >= 6 {
              continue;
              return 10;
            }
          }
        }
      }
    }
  }
  return 0;
}

fn test5() {
  for let i = 0; i < 100; i++ {
    if i == 98 {
      return 100;
    }
  }
  return 0;
}

test {
  test1();
  test2();
  test3();
  assert::eq(test4(), 0);
  assert::eq(test5(), 100);
}
