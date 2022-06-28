fn test1() {
  for let i = 0; i < 1; i++ {
    assert::eq(i, 0);
  }
  assert::eq(try i else 10, 10);

  {
    let i = 0;
    for ; i < 1; i++ {
      assert::eq(i ,0);
    }
    assert::eq(i, 1);
  }

  {
    let i = 0;
    for ; i < 1; {
      i++;
    }
    assert::eq(i, 1);
  }

  {
    let i = 0;
    for ; ; {
      i++;
      if i >= 1 {
        break;
      }
    }
    assert::eq(i, 1);
  }
}

test {
  test1();
}
