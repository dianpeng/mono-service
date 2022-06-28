fn test1() {
  assert::eq((1, 2).first, 1);
  assert::eq((1, 2).second, 2);
  assert::eq((1, 2)[0], 1);
  assert::eq((1, 2)[1], 2);
}

fn testIter() {
  assert::eq(|{
    let i = 0;
    for let _, _ = (1, 2) {
      i++;
    }
    i;
  }, 2);

  {
    let p = (1, 2);
    let index = 0;
    for let k, v = p {
      if index == 0 {
        assert::eq(k, 0);
        assert::eq(v, 1);
      } else {
        assert::eq(k, 1);
        assert::eq(v, 2);
      }
      index++;
    }
    assert::eq(index, 2);
  }
}

test {
  test1();
  testIter();
  assert::eq(type((1, 2)), "pair");
}
