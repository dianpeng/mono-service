fn testInt() {
  assert::eq(1, 1);
  assert::eq(1 + 0, 1);
}

fn testReal() {
  assert::eq(1.0, 1.0);
  assert::eq(1 + 0.0, 1.0);
  assert::eq(1.0 - 1.0, 0.0);
}

fn testNumberToString() {
  assert::eq(1:to_string(), "1");
  assert::eq(1.0:to_string(), "1.000000");
  {
    let x = 1.0/0.0;
    assert::yes(x:is_inf());
    assert::yes(x:is_pinf());
  }
  {
    let yy = -1.0/0.0;
    assert::yes(yy:is_inf());
    assert::yes(yy:is_ninf());
  }
  {
    let z = 1.1;
    assert::eq(z:floor(), 1.0);
  }
  {
    let z = 1.1;
    assert::eq(z:cell(), 2.0);
  }
  assert::eq(type(1), "int");
  assert::eq(type(1.0), "real");
}

fn testBoolean() {
  assert::yes(true);
  assert::no(false);
}

fn testBooleanToString() {
  assert::eq(true:to_string(), "true");
  assert::eq(false:to_string(), "false");
  assert::eq(type(true), "bool");
  assert::eq(type(false), "bool");
}

fn testNull() {
  assert::eq(null:to_string(), "null");
  assert::eq(type(null), "null");
}

test {
  testInt();
  testReal();
  testNumberToString();
  testBooleanToString();
  testNull();
}
