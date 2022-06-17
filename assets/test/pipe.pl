fn xx(a, b, c) {
  return a + b + c;
}

test {
  assert::eq("HELLO", "hello" | str::to_upper);
  assert::eq("HELLO", "hello" | str::to_upper());
  assert::eq(10, 5 | xx(2, 3));
}
