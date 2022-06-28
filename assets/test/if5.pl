fn test1(flag) {
  if flag == 0 {
    return 1;
  } else {
    return 2;
  }
}

fn test2(flag) {
  if flag > 0 {
    if flag >= 2 {
      if flag >= 4 {
        return 1;
      } else {
        return 2;
      }
    } else {
      return 3;
    }
  } else {
    return 4;
  }
}

test {
  assert::eq(test1(0), 1);
  assert::eq(test1(1), 2);
  assert::eq(test2(0), 4);
  assert::eq(test2(1), 3);
  assert::eq(test2(2), 2);
  assert::eq(test2(3), 2);
  assert::eq(test2(4), 1);
}
