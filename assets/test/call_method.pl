/**
 * method call in PL script
 **/

fn test1() {
  assert::eq(1:to_string(), "1");
  {
    let x = 1:to_string;
    assert::eq(x(), "1");
  }

  // capture the method as function
  {
    let y = [];
    let push_back = y:push_back;

    assert::eq(push_back(1), [1]);
    assert::eq(y, [1]);
  }
}

test {
  test1();
}
