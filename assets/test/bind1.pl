fn foo0() {
  return 0;
}

fn foo1(a) {
  return a;
}

fn foo2(a, b) {
  return a + b;
}

fn test1() {
  assert::eq(bind(foo0)(), 0);
  assert::eq(bind(foo1, 1)(), 1);
  assert::eq(bind(foo1, new_placeholder())(11), 11);
}

test {
  test1();
}
